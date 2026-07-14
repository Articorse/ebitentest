package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
	"math/rand/v2"
)

func (cc *ChunkContainer) Tick(
	_ *rand.Rand,
	toBeAdded []utils.Vec2i,
	toBeRemoved []utils.Vec2i,
	required map[utils.Vec2i]struct{},
	_ map[utils.Vec2i]int,
	ecsCont *ecs.ECSContainer,
) error {
	var err error
	cc.currentTickEIdsInChunks, err = cc.getEntityIdsInChunks(ecsCont)
	if err != nil {
		return fmt.Errorf("failed to get entity IDs in chunks: %w", err)
	}

	cc.drainLoadResults(required, ecsCont)
	cc.drainSaveResults(required, ecsCont)
	cc.invalidateNonRequiredLoads(required)

	for _, cPos := range toBeRemoved {
		c, exists := cc.chunks[cPos]
		if !exists {
			continue
		}

		meta := cc.getOrCreateMeta(cPos)
		if meta.state == ChunkState_Evicting {
			continue
		}

		cDto, err := cc.chunkToDto(c, cPos, ecsCont)
		if err != nil {
			return fmt.Errorf("failed to convert chunk to DTO: %w", err)
		}

		eIdsToRemove := make([]common.EntityId, 0, len(cDto.Entities))
		for eId := range cDto.Entities {
			eIdsToRemove = append(eIdsToRemove, eId)
		}
		cc.pendingEvictEntityIDs[cPos] = eIdsToRemove

		cc.setState(cPos, ChunkState_Evicting)
		cc.saveChunkCh <- saveRequest{
			pos:   cPos,
			chunk: cDto,
		}
	}

	dispatched := 0
	for _, cPos := range toBeAdded {
		if dispatched >= data.MaxChunkLoadDispatchPerTick {
			break
		}

		if _, loaded := cc.chunks[cPos]; loaded {
			cc.setState(cPos, ChunkState_Active)
			delete(cc.pendingEvictEntityIDs, cPos)
			continue
		}

		if _, ok := required[cPos]; !ok {
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

	return nil
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

			if err := postLoadChunk(res.chunk, cc.atlas); err != nil {
				log.Printf("Failed post-load of chunk at %v: %v", res.pos, err)
				cc.setState(res.pos, ChunkState_Unloaded)
				continue
			}

			if _, exists := cc.chunks[res.pos]; exists {
				cc.setState(res.pos, ChunkState_Active)
				meta.lastErr = nil
				meta.retries = 0
				continue
			}

			for _, compDtos := range res.ents {
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
					ecsCont.AddEntity(comps...)
				}
			}

			cc.chunks[res.pos] = res.chunk
			meta.lastErr = nil
			meta.retries = 0
			cc.setState(res.pos, ChunkState_Active)
		default:
			return
		}
	}
}

func (cc *ChunkContainer) drainSaveResults(required map[utils.Vec2i]struct{}, ecsCont *ecs.ECSContainer) {
	for {
		select {
		case res := <-cc.saveResultCh:
			meta, ok := cc.chunkMeta[res.pos]
			if !ok {
				delete(cc.pendingEvictEntityIDs, res.pos)
				continue
			}

			if res.err != nil {
				log.Printf("Failed to save chunk at %v: %v", res.pos, res.err)
				meta.lastErr = res.err
				cc.setState(res.pos, ChunkState_Active)
				delete(cc.pendingEvictEntityIDs, res.pos)
				continue
			}

			if meta.state != ChunkState_Evicting {
				delete(cc.pendingEvictEntityIDs, res.pos)
				continue
			}

			if _, stillRequired := required[res.pos]; stillRequired {
				cc.setState(res.pos, ChunkState_Active)
				delete(cc.pendingEvictEntityIDs, res.pos)
				continue
			}

			delete(cc.chunks, res.pos)

			if eIds, ok := cc.pendingEvictEntityIDs[res.pos]; ok {
				for _, eId := range eIds {
					ecsCont.ScheduleRemoveEntity(eId)
				}
			}
			delete(cc.pendingEvictEntityIDs, res.pos)

			cc.setState(res.pos, ChunkState_Unloaded)
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
