package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
	"maps"
	"slices"
)

// Every decision here depends only on loader positions, never on when IO finishes,
// so that the simulation sees the same chunks on the same ticks every run.
func (cc *ChunkContainer) Tick(ecsCont *ecs.ECSContainer) error {
	plan, err := buildChunkPlan(ecsCont)
	if err != nil {
		log.Printf("error building chunk plan: %v", err)
	}

	cc.drainIOResults()

	for _, pos := range plan.activate {
		if _, ok := cc.chunks[pos]; ok {
			continue
		}

		if err := cc.waitUntilAvailable(pos); err != nil {
			return fmt.Errorf("failed to activate chunk at %v: %w", pos, err)
		}

		if err := cc.activateChunk(pos, ecsCont); err != nil {
			return fmt.Errorf("failed to activate chunk at %v: %w", pos, err)
		}
	}

	var deactivating []utils.Vec2i
	for _, pos := range slices.SortedFunc(maps.Keys(cc.chunks), compareChunkPos) {
		if _, ok := plan.keepActive[pos]; !ok {
			deactivating = append(deactivating, pos)
		}
	}

	if err := cc.resolveEntitiesLeavingSimulation(deactivating, ecsCont); err != nil {
		return err
	}

	for _, pos := range deactivating {
		if err := cc.deactivateChunk(pos, ecsCont); err != nil {
			return fmt.Errorf("failed to deactivate chunk at %v: %w", pos, err)
		}
	}

	cc.prefetch(plan)

	return nil
}

// Loads the chunks that will be activated on the first tick, without activating them.
// Returns true once all of them are available, so the first tick doesn't have to wait.
func (cc *ChunkContainer) Prepare(ecsCont *ecs.ECSContainer) (ready bool, err error) {
	plan, err := buildChunkPlan(ecsCont)
	if err != nil {
		return false, fmt.Errorf("error building chunk plan: %w", err)
	}

	cc.drainIOResults()

	ready = true
	for _, pos := range plan.activate {
		if _, ok := cc.chunks[pos]; ok {
			continue
		}

		if cc.isAvailable(pos) {
			continue
		}
		ready = false

		if _, ok := cc.loadsInFlight[pos]; !ok && cc.loadFailures[pos] >= data.ChunkReloadRetries {
			return false, fmt.Errorf("failed to load chunk at %v after %d attempts", pos, cc.loadFailures[pos])
		}

		cc.requestLoad(pos)
	}

	cc.prefetch(plan)

	return ready, nil
}

func (cc *ChunkContainer) prefetch(plan chunkPlan) {
	requested := 0
	for _, pos := range plan.prefetch {
		if requested >= data.MaxChunkLoadDispatchPerTick {
			break
		}

		if _, ok := cc.chunks[pos]; ok {
			continue
		}
		if _, ok := cc.loadsInFlight[pos]; ok {
			continue
		}
		if cc.isAvailable(pos) {
			continue
		}
		if cc.loadFailures[pos] >= data.ChunkReloadRetries {
			continue
		}

		cc.requestLoad(pos)
		requested++
	}

	for pos := range cc.resident {
		if _, ok := plan.resident[pos]; !ok {
			delete(cc.resident, pos)
		}
	}
}

func (cc *ChunkContainer) activateChunk(pos utils.Vec2i, ecsCont *ecs.ECSContainer) error {
	var c *chunk
	var ents map[common.EntityId][]ecs.ComponentDto

	// An evicted copy is newer than anything on disk, since it's only written by the next commit
	if evicted, ok := cc.evictedChunks[pos]; ok {
		c = cc.dtoToChunkData(evicted.dto)
		ents = evicted.dto.Entities
		delete(cc.evictedChunks, pos)
	} else {
		r := cc.resident[pos]
		c = r.chunk
		ents = r.ents
	}
	delete(cc.resident, pos)

	if err := postLoadChunk(c, cc.atlas); err != nil {
		return fmt.Errorf("failed post-load of chunk: %w", err)
	}

	// Insertion order decides the order systems iterate over entities, so it must not depend on map order
	for _, eId := range slices.Sorted(maps.Keys(ents)) {
		instantiateEntity(eId, ents[eId], ecsCont)
	}

	// The queued entities may already have been committed into the file this chunk was loaded from
	queued := cc.queuedEntities[pos]
	for _, eId := range slices.Sorted(maps.Keys(queued)) {
		if !ecsCont.HasEntity(eId) {
			instantiateEntity(eId, queued[eId].dtos, ecsCont)
		}
	}
	delete(cc.queuedEntities, pos)

	cc.chunks[pos] = c

	return nil
}

// Resolves every composition outside the active chunks, including those in chunks about to be deactivated:
// tethered ones move back, persistent ones are stored and everything else is despawned.
func (cc *ChunkContainer) resolveEntitiesLeavingSimulation(deactivating []utils.Vec2i, ecsCont *ecs.ECSContainer) error {
	eIdsInChunks, err := cc.getEntityIdsInChunks(ecsCont)
	if err != nil {
		return fmt.Errorf("failed to get entity IDs in chunks: %w", err)
	}

	for _, pos := range slices.SortedFunc(maps.Keys(eIdsInChunks), compareChunkPos) {
		_, active := cc.chunks[pos]
		leaving := slices.Contains(deactivating, pos)
		if active && !leaving {
			continue
		}

		for _, root := range eIdsInChunks[pos] {
			if cc.relocateTethered(root, ecsCont) {
				continue
			}

			// Stored with the chunk's own snapshot when it's deactivated
			if leaving {
				continue
			}

			if ecsCont.Persistents.HasComponent(root) {
				members, err := ecsCont.GetCompositionComponents(root)
				if err != nil {
					return fmt.Errorf("failed to get composition of entity %d: %w", root, err)
				}
				for _, e := range slices.Sorted(maps.Keys(members)) {
					cc.queueEntity(pos, e, members[e])
				}
			}

			ecsCont.ScheduleRemoveEntity(root)
		}
	}

	// Relocated entities have moved, so this is recomputed for the chunk snapshots that follow
	cc.currentTickEIdsInChunks, err = cc.getEntityIdsInChunks(ecsCont)
	if err != nil {
		return fmt.Errorf("failed to get entity IDs in chunks: %w", err)
	}

	return nil
}

func (cc *ChunkContainer) deactivateChunk(pos utils.Vec2i, ecsCont *ecs.ECSContainer) error {
	c, ok := cc.chunks[pos]
	if !ok {
		return nil
	}

	cDto, err := cc.chunkToDto(c, pos, ecsCont)
	if err != nil {
		return fmt.Errorf("failed to convert chunk to DTO: %w", err)
	}

	for _, root := range cc.currentTickEIdsInChunks[pos] {
		ecsCont.ScheduleRemoveEntity(root)
	}

	cc.pendingVersion++
	cc.evictedChunks[pos] = pendingChunk{dto: cDto, version: cc.pendingVersion}

	c.freeImage()
	delete(cc.chunks, pos)

	return nil
}

func instantiateEntity(eId common.EntityId, compDtos []ecs.ComponentDto, ecsCont *ecs.ECSContainer) {
	comps := make([]ecs.Component, 0, len(compDtos))
	for _, compDto := range compDtos {
		comp, err := ecs.DtoToComponent(compDto)
		if err != nil {
			log.Printf("Failed to convert component DTO to component: %v", err)
			continue
		}
		if comp != nil {
			comps = append(comps, comp)
		}
	}

	if len(comps) > 0 {
		if err := ecsCont.AddEntityWithId(eId, comps...); err != nil {
			log.Printf("Failed to add entity %d with components: %v", eId, err)
		}
	}
}

func compareChunkPos(a, b utils.Vec2i) int {
	if a.X != b.X {
		return a.X - b.X
	}
	return a.Y - b.Y
}
