package tilesystem

import (
	"ebittest/data"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func (cc *ChunkContainer) StartIOThreads() {
	go func() {
		for cDto := range cc.saveChunkCh {
			if err := SaveChunkGob(fmt.Sprintf(data.ChunkDataFilePathTemplate, cDto.X, cDto.Y), cDto); err != nil {
				log.Printf("failed to save chunk at (%d, %d): %v", cDto.X, cDto.Y, err)
			}
		}
	}()

	go func() {
		for ck := range cc.loadChunkCh {
			cDto, exists, err := LoadChunkGob(fmt.Sprintf(data.ChunkDataFilePathTemplate, ck.X, ck.Y))
			if err != nil {
				log.Printf("Failed to load chunk at (%d, %d): %v\n", ck.X, ck.Y, err)
				continue
			}

			var chunk *chunk
			if exists {
				chunk = cc.dtoToChunkData(cDto)
			} else {
				chunk, err = cc.newChunkData(cc.Atlas, ck)
				if err != nil {
					log.Printf("Failed to create new chunk at %v: %v\n", ck, err)
				}
			}

			cc.preloadedChunksMtx.Lock()
			cc.preloadedChunks[ck] = chunk
			if exists {
				cc.preloadedChunkEntities[ck] = cDto.Entities
			}
			cc.preloadedChunksMtx.Unlock()

			cc.dispatchedLoadChunkPositionsMtx.Lock()
			delete(cc.dispatchedLoadChunkPositions, ck)
			cc.dispatchedLoadChunkPositionsMtx.Unlock()
		}
	}()
}

func SaveChunkGob(filePath string, data chunkDto) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	encoder := gob.NewEncoder(file)

	err = encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("error closing file: %w", err)
	}

	return nil
}

func LoadChunkGob(filePath string) (cDto chunkDto, exists bool, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return chunkDto{}, false, nil
		}
		return chunkDto{}, false, fmt.Errorf("error opening file: %w", err)
	}

	var data chunkDto
	decoder := gob.NewDecoder(file)

	err = decoder.Decode(&data)
	if err != nil {
		return chunkDto{}, true, fmt.Errorf("error decoding data: %w", err)
	}

	err = file.Close()
	if err != nil {
		return chunkDto{}, true, fmt.Errorf("error closing file: %w", err)
	}

	return data, true, nil
}

func PostLoadChunk(c *chunk, atlas map[data.TileEnum]TileDef) error {
	if err := c.generateChunkImage(atlas); err != nil {
		return fmt.Errorf("failed to generate chunk image: %w", err)
	}

	return nil
}
