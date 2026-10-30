package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type chunkLoadMeta struct {
	state   chunkState
	retries int8
	lastErr error
	seq     uint64
}

type loadRequest struct {
	pos utils.Vec2i
	seq uint64
}

type saveRequest struct {
	pos   utils.Vec2i
	chunk chunkDto
}

type loadResult struct {
	pos   utils.Vec2i
	seq   uint64
	chunk *chunk
	ents  map[common.EntityId][]ecs.ComponentDto
	err   error
}

type saveResult struct {
	pos utils.Vec2i
	err error
}

func (cc *ChunkContainer) StartIOThreads() {
	go func() {
		for req := range cc.saveChunkCh {
			err := saveChunkGob(req.pos, req.chunk)
			cc.saveResultCh <- saveResult{
				pos: req.pos,
				err: err,
			}
		}
	}()

	go func() {
		for req := range cc.loadReqCh {
			cDto, exists, err := loadChunkGob(req.pos)
			if err != nil {
				cc.loadResultCh <- loadResult{
					pos: req.pos,
					seq: req.seq,
					err: err,
				}
				continue
			}

			var loadedChunk *chunk
			var ents map[common.EntityId][]ecs.ComponentDto

			if exists {
				loadedChunk = cc.dtoToChunkData(cDto)
				ents = cDto.Entities
			} else {
				loadedChunk, err = cc.newChunkData(req.pos)
				if err != nil {
					cc.loadResultCh <- loadResult{
						pos: req.pos,
						seq: req.seq,
						err: fmt.Errorf("failed to create new chunk: %w", err),
					}
					continue
				}
			}

			cc.loadResultCh <- loadResult{
				pos:   req.pos,
				seq:   req.seq,
				chunk: loadedChunk,
				ents:  ents,
			}
		}
	}()
}

func saveChunkGob(cellKey utils.Vec2i, chunkData chunkDto) (err error) {
	filePath := fmt.Sprintf(data.ChunkDataFilePathTemplate, cellKey.X, cellKey.Y)
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close file: %w", cerr))
		}
	}()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(chunkData); err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	return nil
}

func loadChunkGob(cellKey utils.Vec2i) (cDto chunkDto, exists bool, err error) {
	file, err := os.Open(fmt.Sprintf(data.ChunkDataFilePathTemplate, cellKey.X, cellKey.Y))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return chunkDto{}, false, nil
		}
		return chunkDto{}, false, fmt.Errorf("error opening file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close file: %w", cerr))
		}
	}()

	var data chunkDto
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return chunkDto{}, true, fmt.Errorf("error decoding data: %w", err)
	}

	return data, true, nil
}

func postLoadChunk(c *chunk, atlas map[data.TileEnum]tileDef) error {
	for _, tEnum := range c.tiles {
		if _, ok := atlas[tEnum]; !ok {
			return fmt.Errorf("tile enum %d not found in atlas", tEnum)
		}
	}

	return nil
}

func (cc *ChunkContainer) enqueueLoadRequest(req loadRequest) bool {
	select {
	case cc.loadReqCh <- req:
		return true
	default:
		return false
	}
}

func (cc *ChunkContainer) getOrCreateMeta(pos utils.Vec2i) *chunkLoadMeta {
	meta, ok := cc.chunkMeta[pos]
	if ok {
		return meta
	}

	meta = &chunkLoadMeta{state: ChunkState_Unloaded}
	cc.chunkMeta[pos] = meta
	return meta
}

func (cc *ChunkContainer) saveQueuedEntitiesToGob(
	queuedEntities map[utils.Vec2i]map[common.EntityId][]ecs.ComponentDto,
	ecsCont *ecs.ECSContainer,
) (failedEntities map[utils.Vec2i]map[common.EntityId][]ecs.ComponentDto, err error) {
	failedEntities = make(map[utils.Vec2i]map[common.EntityId][]ecs.ComponentDto)

	for ck := range queuedEntities {
		cDto, exists, err := loadChunkGob(ck)
		if err != nil {
			failedEntities[ck] = queuedEntities[ck]
			log.Printf("Failed to load chunk for saving queued entities: %v", err)
			continue
		}

		if !exists {
			newChunk, err := cc.newChunkData(ck)
			if err != nil {
				failedEntities[ck] = queuedEntities[ck]
				log.Printf("Failed to create new chunk for saving queued entities: %v", err)
				continue
			}

			cDto, err = cc.chunkToDto(newChunk, ck, ecsCont)
			if err != nil {
				failedEntities[ck] = queuedEntities[ck]
				log.Printf("Failed to convert new chunk to DTO for saving queued entities: %v", err)
				continue
			}
		}

		for eId := range queuedEntities[ck] {
			if cDto.Entities == nil {
				cDto.Entities = make(map[common.EntityId][]ecs.ComponentDto)
			}

			if _, ok := cDto.Entities[eId]; !ok {
				cDto.Entities[eId] = queuedEntities[ck][eId]
			}
		}

		err = saveChunkGob(ck, cDto)
		if err != nil {
			failedEntities[ck] = queuedEntities[ck]
			log.Printf("Failed to save chunk with queued entities: %v", err)
			continue
		}
	}

	return failedEntities, nil
}

func (cc *ChunkContainer) SaveAllAndWait(ecsCont *ecs.ECSContainer) error {
	var errs error

	eIdsInChunks, _, err := cc.getEntityIdsInChunks(ecsCont)
	if err != nil {
		return fmt.Errorf("failed to get entity IDs in chunks: %w", err)
	}
	cc.currentTickEIdsInChunks = eIdsInChunks

	receiveSaveResult := func(res saveResult) {
		cc.pendingSaves--
		if res.err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to save chunk at %v: %w", res.pos, res.err))
		}
	}

	for cPos, c := range cc.chunks {
		cDto, err := cc.chunkToDto(c, cPos, ecsCont)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to convert chunk at %v to DTO: %w", cPos, err))
			continue
		}

		req := saveRequest{pos: cPos, chunk: cDto}

		// Results are received while sending, since the IO thread blocks once the result channel is full
		for sent := false; !sent; {
			select {
			case cc.saveChunkCh <- req:
				cc.pendingSaves++
				sent = true
			case res := <-cc.saveResultCh:
				receiveSaveResult(res)
			}
		}
	}

	for cc.pendingSaves > 0 {
		receiveSaveResult(<-cc.saveResultCh)
	}

	cc.queuedUnloadMtx.Lock()
	queuedUnload := cc.queuedUnloadEntityIDs
	cc.queuedUnloadEntityIDs = make(map[utils.Vec2i]map[common.EntityId][]ecs.ComponentDto)
	cc.queuedUnloadMtx.Unlock()

	failedEntities, err := cc.saveQueuedEntitiesToGob(queuedUnload, ecsCont)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to save queued entities: %w", err))
	}
	if len(failedEntities) > 0 {
		errs = errors.Join(errs, fmt.Errorf("failed to save queued entities of %d chunks", len(failedEntities)))
	}

	return errs
}
