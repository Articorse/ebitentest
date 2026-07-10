package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"sync"
)

type ChunkContainer struct {
	Chunks map[utils.CellKey]*chunk
	Atlas  map[data.TileEnum]TileDef

	saveChunkCh chan chunkDto
	loadChunkCh chan utils.CellKey

	dispatchedLoadChunkPositionsMtx sync.Mutex
	dispatchedLoadChunkPositions    map[utils.CellKey]struct{}

	preloadedChunksMtx     sync.Mutex
	preloadedChunks        map[utils.CellKey]*chunk
	preloadedChunkEntities map[utils.CellKey]map[common.EntityId][]ecs.ComponentDto
}

func NewChunkContainer() *ChunkContainer {
	cc := &ChunkContainer{
		Chunks: make(map[utils.CellKey]*chunk),

		saveChunkCh:                  make(chan chunkDto),
		loadChunkCh:                  make(chan utils.CellKey),
		dispatchedLoadChunkPositions: make(map[utils.CellKey]struct{}),
	}

	cc.StartIOThreads()

	return cc
}

func (cc *ChunkContainer) newChunkData(
	atlas map[data.TileEnum]TileDef,
	pos utils.CellKey,
) (*chunk, error) {
	r := rand.NewPCG(data.RngSeed1+uint64(pos.X), data.RngSeed2+uint64(pos.Y))

	c := chunk{}
	c.promotedTiles = make(map[utils.CellKey]promotedTile)

	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			tileId := data.TileEnum(r.Uint64()%3 + 1)
			c.tiles[y*data.ChunkSize+x] = tileId
		}
	}

	return &c, nil
}

func (cc *ChunkContainer) Tick(
	r *rand.Rand,
	toBeAdded []utils.CellKey,
	toBeRemoved []utils.CellKey,
	ecsCont *ecs.ECSContainer,
) error {
	cc.preloadedChunksMtx.Lock()
	preloadedChunksCopy := make(map[utils.CellKey]*chunk)
	for k, v := range cc.preloadedChunks {
		preloadedChunksCopy[k] = v
	}
	cc.preloadedChunks = make(map[utils.CellKey]*chunk)

	preloadedChunkEntitiesCopy := make(map[utils.CellKey]map[common.EntityId][]ecs.ComponentDto)
	for k, v := range cc.preloadedChunkEntities {
		preloadedChunkEntitiesCopy[k] = v
	}
	cc.preloadedChunkEntities = make(map[utils.CellKey]map[common.EntityId][]ecs.ComponentDto)
	cc.preloadedChunksMtx.Unlock()

	for cPos, c := range preloadedChunksCopy {
		if _, exists := cc.Chunks[cPos]; exists {
			continue
		}

		err := c.generateChunkImage(cc.Atlas)
		if err != nil {
			log.Printf("Failed to generate chunk image for chunk at %v: %v\n", cPos, err)
			continue
		}

		entities, ok := preloadedChunkEntitiesCopy[cPos]
		if ok {
			for _, compDtos := range entities {
				comps := make([]ecs.Component, len(compDtos))
				for i, compDto := range compDtos {
					comp, err := ecs.DtoToComponent(compDto)
					if err != nil {
						log.Printf("Failed to convert component DTO to component: %v\n", err)
						continue
					}
					comps[i] = comp
				}
				ecsCont.AddEntity(comps...)
			}
		}

		cc.Chunks[cPos] = c
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

		cc.saveChunkCh <- cDto

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
		if _, loaded := cc.Chunks[cPos]; loaded {
			continue
		}

		cc.dispatchedLoadChunkPositionsMtx.Lock()
		if _, dispatched := cc.dispatchedLoadChunkPositions[cPos]; dispatched {
			cc.dispatchedLoadChunkPositionsMtx.Unlock()
			continue
		}

		cc.dispatchedLoadChunkPositions[cPos] = struct{}{}
		cc.dispatchedLoadChunkPositionsMtx.Unlock()

		cc.loadChunkCh <- cPos
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
		chunk := cc.dtoToChunkData(dto)

		err := chunk.generateChunkImage(cc.Atlas)
		if err != nil {
			fmt.Printf("error generating chunk image for chunk at (%d, %d): %v\n", dto.X, dto.Y, err)
			continue
		}

		chunks[utils.CellKey{X: dto.X, Y: dto.Y}] = chunk
	}

	return chunks
}

func (*ChunkContainer) dtoToChunkData(dto chunkDto) *chunk {
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
	return chunk
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
