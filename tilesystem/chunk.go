package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

type chunk struct {
	pos           utils.Vec2
	tiles         [data.ChunkSize * data.ChunkSize]data.TileEnum
	promotedTiles map[common.CellKey]promotedTile

	Image *ebiten.Image
}

func (x *chunk) GetPos() utils.Vec2 {
	return x.pos
}

func (x *chunk) GetTileDefId(cellKey common.CellKey) data.TileEnum {
	if cellKey.X < 0 || cellKey.X >= data.ChunkSize || cellKey.Y < 0 || cellKey.Y >= data.ChunkSize {
		return 0
	}

	localX := ((cellKey.X % data.ChunkSize) + data.ChunkSize) % data.ChunkSize
	localY := ((cellKey.Y % data.ChunkSize) + data.ChunkSize) % data.ChunkSize
	idx := localY*data.ChunkSize + localX
	if len(x.tiles) <= idx {
		return 0
	}

	return x.tiles[idx]
}

func (x *chunk) GetPromotedTiles() map[common.CellKey]promotedTile {
	return x.promotedTiles
}

func (x *chunk) generateChunkImage(atlas map[data.TileEnum]TileDef) error {
	x.Image = ebiten.NewImage(data.ChunkSize*data.TileSize, data.ChunkSize*data.TileSize)

	for tCell, tEnum := range x.tiles {
		tile, ok := atlas[tEnum]
		if !ok {
			return fmt.Errorf("tile enum %d not found in atlas", tEnum)
		}

		opts := ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(tCell%data.ChunkSize*data.TileSize), float64(tCell/data.ChunkSize*data.TileSize))
		x.Image.DrawImage(tile.Image, &opts)
	}

	return nil
}
