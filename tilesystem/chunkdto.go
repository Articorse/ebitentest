package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"maps"
)

type chunkDto struct {
	X               int
	Y               int
	Tiles           [data.ChunkSize * data.ChunkSize]data.TileEnum
	PromotedTiles   []promotedTileDto
	Entities        map[common.EntityId][]ecs.ComponentDto
	TerrainModified bool
}

type promotedTileDto struct {
	X             int
	Y             int
	CurrentHealth int
}

func (cc *ChunkContainer) chunkToDto(
	c *chunk,
	pos utils.Vec2i,
	ecsCont *ecs.ECSContainer,
) (chunkDto, error) {
	promotedTiles := make([]promotedTileDto, 0, len(c.promotedTiles))
	for pPos, pTile := range c.promotedTiles {
		promotedTiles = append(promotedTiles, promotedTileDto{
			X:             pPos.X,
			Y:             pPos.Y,
			CurrentHealth: pTile.currentHealth,
		})
	}

	entitiesWithComps := make(map[common.EntityId][]ecs.ComponentDto)
	for _, root := range cc.currentTickEIdsInChunks[pos] {
		if !ecsCont.Persistents.HasComponent(root) {
			continue
		}

		members, err := ecsCont.GetCompositionComponents(root)
		if err != nil {
			return chunkDto{}, fmt.Errorf("failed to get composition of entity %d: %w", root, err)
		}
		maps.Copy(entitiesWithComps, members)
	}

	return chunkDto{
		X:               pos.X,
		Y:               pos.Y,
		Tiles:           c.tiles,
		PromotedTiles:   promotedTiles,
		Entities:        entitiesWithComps,
		TerrainModified: c.terrainModified,
	}, nil
}

func (*ChunkContainer) dtoToChunkData(dto chunkDto) *chunk {
	promotedTiles := make(map[utils.Vec2i]promotedTile)
	for _, pDto := range dto.PromotedTiles {
		promotedTiles[utils.Vec2i{X: pDto.X, Y: pDto.Y}] = promotedTile{
			currentHealth: pDto.CurrentHealth,
		}
	}

	chunk := &chunk{
		tiles:           dto.Tiles,
		promotedTiles:   promotedTiles,
		terrainModified: dto.TerrainModified,
	}
	return chunk
}

func (dto chunkDto) needsSaving() bool {
	return dto.TerrainModified || len(dto.PromotedTiles) > 0 || len(dto.Entities) > 0
}
