package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
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

type loadResult struct {
	pos   utils.Vec2i
	seq   uint64
	chunk *chunk
	ents  map[common.EntityId][]ecs.ComponentDto
	err   error
}

func (cc *ChunkContainer) StartIOThreads() {
	go func() {
		for req := range cc.commitReqCh {
			committed, err := cc.commit(req)
			cc.commitResultCh <- commitResult{committed: committed, err: err}
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
