package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
)

func (cc *ChunkContainer) Tick(
	toBeAdded []utils.Vec2i,
	toBeRemoved []utils.Vec2i,
	required map[utils.Vec2i]struct{},
	ecsCont *ecs.ECSContainer,
) error {
	eIdsInChunks, toBeUnloaded, err := cc.getEntityIdsInChunks(ecsCont)
	if err != nil {
		return fmt.Errorf("failed to get entity IDs in chunks: %w", err)
	}
	cc.currentTickEIdsInChunks = eIdsInChunks

	for ck, eIds := range toBeUnloaded {
		for _, eId := range eIds {
			cc.queueEntity(ck, eId, ecsCont.GetEntityComponents(eId))
			ecsCont.ScheduleRemoveEntity(eId)
		}
	}

	cc.drainLoadResults(required, ecsCont)
	cc.drainCommitResults()
	cc.invalidateNonRequiredLoads(required)

	for _, cPos := range toBeRemoved {
		if err := cc.evictChunk(cPos, ecsCont); err != nil {
			return fmt.Errorf("failed to evict chunk at %v: %w", cPos, err)
		}
	}

	cc.dispatchLoads(toBeAdded, required, ecsCont)

	return nil
}

func (cc *ChunkContainer) TickLoading(
	toBeAdded []utils.Vec2i,
	required map[utils.Vec2i]struct{},
	ecsCont *ecs.ECSContainer,
) (loaded bool) {
	cc.drainLoadResults(required, ecsCont)
	cc.invalidateNonRequiredLoads(required)
	cc.dispatchLoads(toBeAdded, required, ecsCont)

	for pos := range required {
		if _, ok := cc.chunks[pos]; !ok {
			return false
		}
	}

	return true
}

func (cc *ChunkContainer) evictChunk(pos utils.Vec2i, ecsCont *ecs.ECSContainer) error {
	c, ok := cc.chunks[pos]
	if !ok {
		return nil
	}

	cDto, err := cc.chunkToDto(c, pos, ecsCont)
	if err != nil {
		return fmt.Errorf("failed to convert chunk to DTO: %w", err)
	}

	for eId := range cDto.Entities {
		ecsCont.ScheduleRemoveEntity(eId)
	}

	cc.pendingVersion++
	cc.evictedChunks[pos] = pendingChunk{dto: cDto, version: cc.pendingVersion}

	c.freeImage()
	delete(cc.chunks, pos)
	cc.setState(pos, ChunkState_Unloaded)

	return nil
}

func (cc *ChunkContainer) activateChunk(
	pos utils.Vec2i,
	c *chunk,
	ents map[common.EntityId][]ecs.ComponentDto,
	ecsCont *ecs.ECSContainer,
) error {
	if err := postLoadChunk(c, cc.atlas); err != nil {
		return fmt.Errorf("failed post-load of chunk: %w", err)
	}

	for eId, compDtos := range ents {
		instantiateEntity(eId, compDtos, ecsCont)
	}

	// The queued entities may already have been committed into the file this chunk was loaded from
	for eId, queued := range cc.queuedEntities[pos] {
		if !ecsCont.HasEntity(eId) {
			instantiateEntity(eId, queued.dtos, ecsCont)
		}
	}
	delete(cc.queuedEntities, pos)

	cc.chunks[pos] = c
	cc.setState(pos, ChunkState_Active)

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

func (cc *ChunkContainer) drainLoadResults(required map[utils.Vec2i]struct{}, ecsCont *ecs.ECSContainer) {
	for {
		select {
		case res := <-cc.loadResultCh:
			meta, ok := cc.chunkMeta[res.pos]
			if !ok {
				continue
			}

			if res.seq != meta.seq {
				continue
			}

			if res.err != nil {
				meta.lastErr = res.err
				meta.retries++

				log.Printf("Failed to load chunk at %v (retry %d): %v", res.pos, meta.retries, res.err)

				if meta.retries < data.ChunkReloadRetries {
					meta.seq++
					retrySeq := meta.seq
					if cc.enqueueLoadRequest(loadRequest{pos: res.pos, seq: retrySeq}) {
						cc.setState(res.pos, ChunkState_Loading)
					} else {
						cc.setState(res.pos, ChunkState_Unloaded)
					}
				} else {
					cc.setState(res.pos, ChunkState_Unloaded)
				}
				continue
			}

			if res.chunk == nil {
				log.Printf("Loaded chunk at %v is nil", res.pos)
				cc.setState(res.pos, ChunkState_Unloaded)
				continue
			}

			if _, ok := required[res.pos]; !ok {
				cc.setState(res.pos, ChunkState_Unloaded)
				continue
			}

			if _, exists := cc.chunks[res.pos]; exists {
				cc.setState(res.pos, ChunkState_Active)
				meta.lastErr = nil
				meta.retries = 0
				continue
			}

			if err := cc.activateChunk(res.pos, res.chunk, res.ents, ecsCont); err != nil {
				log.Printf("Failed to activate chunk at %v: %v", res.pos, err)
				cc.setState(res.pos, ChunkState_Unloaded)
				continue
			}

			meta.lastErr = nil
			meta.retries = 0
		default:
			return
		}
	}
}

func (cc *ChunkContainer) invalidateNonRequiredLoads(required map[utils.Vec2i]struct{}) {
	for pos, meta := range cc.chunkMeta {
		if _, ok := required[pos]; ok {
			continue
		}

		if meta.state == ChunkState_Loading {
			meta.seq++
			cc.setState(pos, ChunkState_Unloaded)
		}
	}
}

func (cc *ChunkContainer) dispatchLoads(
	toBeAdded []utils.Vec2i,
	required map[utils.Vec2i]struct{},
	ecsCont *ecs.ECSContainer,
) {
	dispatched := 0
	for _, cPos := range toBeAdded {
		if dispatched >= data.MaxChunkLoadDispatchPerTick {
			break
		}

		if _, loaded := cc.chunks[cPos]; loaded {
			cc.setState(cPos, ChunkState_Active)
			continue
		}

		if _, ok := required[cPos]; !ok {
			continue
		}

		// The file on disk is older than the evicted copy, which is only written by the next commit
		if evicted, ok := cc.evictedChunks[cPos]; ok {
			delete(cc.evictedChunks, cPos)
			if err := cc.activateChunk(cPos, cc.dtoToChunkData(evicted.dto), evicted.dto.Entities, ecsCont); err != nil {
				log.Printf("Failed to activate evicted chunk at %v: %v", cPos, err)
			}
			continue
		}

		meta := cc.getOrCreateMeta(cPos)
		if meta.state == ChunkState_Loading {
			continue
		}

		meta.seq++
		seq := meta.seq

		if !cc.enqueueLoadRequest(loadRequest{pos: cPos, seq: seq}) {
			break
		}

		cc.setState(cPos, ChunkState_Loading)
		dispatched++
	}
}
