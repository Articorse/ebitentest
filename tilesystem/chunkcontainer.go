package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
)

type ChunkContainer struct {
	Chunks map[utils.CellKey]*chunk
	Atlas  map[data.TileEnum]TileDef
}

func (cc *ChunkContainer) newChunk(
	r *rand.Rand,
	atlas map[data.TileEnum]TileDef,
	pos utils.CellKey,
	ecsCont *ecs.ECSContainer,
) (*chunk, error) {
	c := chunk{}
	c.promotedTiles = make(map[utils.CellKey]promotedTile)

	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			tileId := data.TileEnum(r.IntN(3) + 1)
			c.tiles[y*data.ChunkSize+x] = tileId
		}
	}

	cDto, err := cc.ChunkToDto(&c, pos, ecsCont)
	if err != nil {
		return nil, fmt.Errorf("failed to convert chunk to DTO: %w", err)
	}
	if err := SaveChunkGob(fmt.Sprintf(data.ChunkDataFilePathTemplate, cDto.X, cDto.Y), cDto); err != nil {
		return nil, fmt.Errorf("failed to save new chunk: %w", err)
	}

	if err := c.generateChunkImage(atlas); err != nil {
		return nil, fmt.Errorf("failed to generate chunk image: %w", err)
	}

	return &c, nil
}

func (cc *ChunkContainer) Tick(
	r *rand.Rand,
	toBeAdded []utils.CellKey,
	toBeRemoved []utils.CellKey,
	ecsCont *ecs.ECSContainer,
) error {
	if cc.Chunks == nil {
		cc.Chunks = make(map[utils.CellKey]*chunk)
	}

	for _, cPos := range toBeRemoved {
		c, exists := cc.Chunks[cPos]
		if !exists {
			continue
		}

		cDto, err := cc.ChunkToDto(c, cPos, ecsCont)
		if err != nil {
			return fmt.Errorf("failed to convert chunk to DTO: %w", err)
		}

		fileName := fmt.Sprintf(data.ChunkDataFilePathTemplate, cDto.X, cDto.Y)
		if err := SaveChunkGob(fileName, cDto); err != nil {
			return fmt.Errorf("failed to save chunk at %v: %w", cPos, err)
		}

		delete(cc.Chunks, cPos)
		entityIds, err := cc.GetEntityIdsInChunk(cPos, ecsCont)
		if err != nil {
			return fmt.Errorf("failed to get entity ids in chunk at %v: %w", cPos, err)
		}

		for _, eId := range entityIds {
			ecsCont.ScheduleRemoveEntity(eId)
		}
	}
	for _, cPos := range toBeAdded {
		fileName := fmt.Sprintf(data.ChunkDataFilePathTemplate, cPos.X, cPos.Y)
		cDto, err := LoadChunkGob(fileName)
		if err == nil {
			chunkMap := cc.DtosToChunks([]chunkDto{cDto})
			cc.Chunks[cPos] = chunkMap[cPos]
			for _, cDtos := range cDto.Entities {
				comps := make([]ecs.Component, len(cDtos))
				for i, cDto := range cDtos {
					comp, err := ecs.DtoToComponent(cDto)
					if err != nil {
						return fmt.Errorf("failed to convert component DTO to component: %w", err)
					}
					comps[i] = comp
				}
				ecsCont.AddEntity(comps...)
			}
		} else if errors.Is(err, os.ErrNotExist) {
			c, err := cc.newChunk(r, cc.Atlas, cPos, ecsCont)
			if err != nil {
				return fmt.Errorf("failed to create new chunk at %v: %w", cPos, err)
			}
			cc.Chunks[cPos] = c
		} else {
			return fmt.Errorf("failed to load chunk at %v: %w", cPos, err)
		}
	}
	return nil
}

func (cc *ChunkContainer) GenerateTileAtlasFromJson(input string) error {
	f, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("error reading tile atlas json file: %w", err)
	}

	var tileDefs *[]tileDefDTO
	err = json.Unmarshal(f, &tileDefs)
	if err != nil {
		return fmt.Errorf("error unmarshalling tile atlas json: %w", err)
	}

	atlas, err := dtosToDefsMap(*tileDefs)
	if err != nil {
		return fmt.Errorf("error converting tile atlas DTOs to defs: %w", err)
	}

	cc.Atlas = atlas

	return nil
}

func (cc *ChunkContainer) ChunkToDto(c *chunk, pos utils.CellKey, ecsCont *ecs.ECSContainer) (chunkDto, error) {
	promotedTiles := make([]promotedTileDto, 0, len(c.promotedTiles))
	for pPos, pTile := range c.promotedTiles {
		promotedTiles = append(promotedTiles, promotedTileDto{
			X:             pPos.X,
			Y:             pPos.Y,
			CurrentHealth: pTile.currentHealth,
		})
	}

	entityIds, err := cc.GetEntityIdsInChunk(pos, ecsCont)
	if err != nil {
		return chunkDto{}, fmt.Errorf("failed to get entity ids in chunk at %v: %w", pos, err)
	}

	entities := ecsCont.GetEntitiesWithComponents(entityIds)

	return chunkDto{
		X:             pos.X,
		Y:             pos.Y,
		Tiles:         c.tiles,
		PromotedTiles: promotedTiles,
		Entities:      entities,
	}, nil
}

func (cc *ChunkContainer) DtosToChunks(dtos []chunkDto) map[utils.CellKey]*chunk {
	chunks := make(map[utils.CellKey]*chunk)
	for _, dto := range dtos {
		promotedTiles := make(map[utils.CellKey]promotedTile)
		for _, pDto := range dto.PromotedTiles {
			promotedTiles[utils.CellKey{X: pDto.X, Y: pDto.Y}] = promotedTile{
				currentHealth: pDto.CurrentHealth,
			}
		}

		chunk := &chunk{
			tiles:         dto.Tiles,
			promotedTiles: promotedTiles,
		}

		err := chunk.generateChunkImage(cc.Atlas)
		if err != nil {
			fmt.Printf("error generating chunk image for chunk at (%d, %d): %v\n", dto.X, dto.Y, err)
			continue
		}

		chunks[utils.CellKey{X: dto.X, Y: dto.Y}] = chunk
	}

	return chunks
}

func (cc *ChunkContainer) GetEntityIdsInChunk(cPos utils.CellKey, ecsCont *ecs.ECSContainer) ([]common.EntityId, error) {
	tm := ecsCont.TransformManager

	eIds := make([]common.EntityId, 0)
	for _, e := range ecsCont.Transforms.GetEntities() {
		eWorldPos, err := tm.GetWorldPos(e, ecsCont)
		if err != nil {
			return nil, fmt.Errorf("error getting world position of entity %d: %v", e, err)
		}

		eChunkPos := utils.WorldPosToChunkGridPos(eWorldPos)
		if eChunkPos == cPos {
			eIds = append(eIds, e)
		}
	}

	return eIds, nil
}

func SaveChunkGob(filePath string, data chunkDto) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	file, _ := os.Create(filePath)
	encoder := gob.NewEncoder(file)

	err := encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("error closing file: %w", err)
	}

	return nil
}

func LoadChunkGob(filePath string) (chunkDto, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return chunkDto{}, fmt.Errorf("error opening file: %w", err)
	}

	var data chunkDto
	decoder := gob.NewDecoder(file)

	err = decoder.Decode(&data)
	if err != nil {
		return chunkDto{}, fmt.Errorf("error decoding data: %w", err)
	}

	err = file.Close()
	if err != nil {
		return chunkDto{}, fmt.Errorf("error closing file: %w", err)
	}

	return data, nil
}
