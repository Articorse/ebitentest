package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
)

type chunkDto struct {
	X             int
	Y             int
	Tiles         [data.ChunkSize * data.ChunkSize]data.TileEnum
	PromotedTiles []promotedTileDto
	Entities      map[common.EntityId][]ecs.ComponentDto
}

type promotedTileDto struct {
	X             int
	Y             int
	CurrentHealth int
}

func (cc *ChunkContainer) chunkToDto(
	c *chunk,
	pos utils.CellKey,
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
	eIds, ok := cc.currentTickEIdsInChunks[pos]
	if ok {
		for _, e := range eIds {
			entitiesWithComps[e] = ecsCont.GetEntityComponents(e)
		}
	}

	return chunkDto{
		X:             pos.X,
		Y:             pos.Y,
		Tiles:         c.tiles,
		PromotedTiles: promotedTiles,
		Entities:      entitiesWithComps,
	}, nil
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
