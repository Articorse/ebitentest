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
	"path/filepath"
)

type chunkLoadMeta struct {
	state   chunkState
	retries int8
	lastErr error
	seq     uint64
}

type loadRequest struct {
	pos utils.CellKey
	seq uint64
}

type saveRequest struct {
	pos   utils.CellKey
	chunk chunkDto
}

type loadResult struct {
	pos   utils.CellKey
	seq   uint64
	chunk *chunk
	ents  map[common.EntityId][]ecs.ComponentDto
	err   error
}

type saveResult struct {
	pos utils.CellKey
	err error
}

func (cc *ChunkContainer) StartIOThreads() {
	go func() {
		for req := range cc.saveChunkCh {
			err := saveChunkGob(fmt.Sprintf(data.ChunkDataFilePathTemplate, req.pos.X, req.pos.Y), req.chunk)
			cc.saveResultCh <- saveResult{
				pos: req.pos,
				err: err,
			}
		}
	}()

	go func() {
		for req := range cc.loadReqCh {
			cDto, exists, err := loadChunkGob(fmt.Sprintf(data.ChunkDataFilePathTemplate, req.pos.X, req.pos.Y))
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

func saveChunkGob(filePath string, data chunkDto) (err error) {
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
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	return nil
}

func loadChunkGob(filePath string) (cDto chunkDto, exists bool, err error) {
	file, err := os.Open(filePath)
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
	if err := c.generateChunkImage(atlas); err != nil {
		return fmt.Errorf("failed to generate chunk image: %w", err)
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

func (cc *ChunkContainer) getOrCreateMeta(pos utils.CellKey) *chunkLoadMeta {
	meta, ok := cc.chunkMeta[pos]
	if ok {
		return meta
	}

	meta = &chunkLoadMeta{state: ChunkState_Unloaded}
	cc.chunkMeta[pos] = meta
	return meta
}
