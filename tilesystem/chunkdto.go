package tilesystem

import (
	"cmp"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"maps"
	"slices"
)

type chunkDto struct {
	X               int
	Y               int
	Tiles           [data.ChunkSize * data.ChunkSize]data.TileEnum
	PromotedTiles   []promotedTileDto
	Entities        map[common.EntityId][]ecs.ComponentDto
	TerrainModified bool
	Features        []featureDto
}

type featureDto struct {
	X       int
	Y       int
	Feature data.FeatureEnum
	// 0 for an undamaged feature
	Health int
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
	// Sorted, so that saving the same chunk twice writes the same file
	slices.SortFunc(promotedTiles, func(a, b promotedTileDto) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})

	features := make([]featureDto, 0, len(c.features))
	for t, f := range c.features {
		features = append(features, featureDto{X: t.X, Y: t.Y, Feature: f, Health: c.featureHealth[t]})
	}
	slices.SortFunc(features, func(a, b featureDto) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})

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
		Features:        features,
	}, nil
}

func (*ChunkContainer) dtoToChunkData(dto chunkDto) *chunk {
	promotedTiles := make(map[utils.Vec2i]promotedTile)
	for _, pDto := range dto.PromotedTiles {
		promotedTiles[utils.Vec2i{X: pDto.X, Y: pDto.Y}] = promotedTile{
			currentHealth: pDto.CurrentHealth,
		}
	}

	features := make(map[utils.Vec2i]data.FeatureEnum, len(dto.Features))
	featureHealth := make(map[utils.Vec2i]int)
	for _, f := range dto.Features {
		t := utils.Vec2i{X: f.X, Y: f.Y}
		features[t] = f.Feature
		if f.Health > 0 {
			featureHealth[t] = f.Health
		}
	}

	chunk := &chunk{
		tiles:           dto.Tiles,
		promotedTiles:   promotedTiles,
		features:        features,
		featureHealth:   featureHealth,
		terrainModified: dto.TerrainModified,
	}
	return chunk
}

func (dto chunkDto) needsSaving() bool {
	return dto.TerrainModified || len(dto.PromotedTiles) > 0 || len(dto.Entities) > 0
}
