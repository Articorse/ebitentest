package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

type chunk struct {
	pos           utils.Vec2
	tiles         map[common.CellKey]data.TileEnum
	promotedTiles map[common.CellKey]promotedTile

	Image *ebiten.Image
}

func (c *chunk) newChunk(r *rand.Rand, atlas map[data.TileEnum]TileDef) error {
	c.tiles = make(map[common.CellKey]data.TileEnum)
	c.promotedTiles = make(map[common.CellKey]promotedTile)

	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			tileId := data.TileEnum(r.IntN(3) + 1)
			c.tiles[common.CellKey{X: x, Y: y}] = tileId
		}
	}

	if err := c.generateChunkImage(atlas); err != nil {
		return fmt.Errorf("failed to generate chunk image: %v", err)
	}

	return nil
}

func (x chunk) GetPos() utils.Vec2 {
	return x.pos
}

func (x chunk) GetTiles() map[common.CellKey]data.TileEnum {
	return x.tiles
}

func (x chunk) GetTileDefId(cellKey common.CellKey) (data.TileEnum, bool) {
	tile, exists := x.tiles[cellKey]
	return tile, exists
}

func (x chunk) GetPromotedTiles() map[common.CellKey]promotedTile {
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
		opts.GeoM.Translate(float64(tCell.X*data.TileSize)-data.TileSize/2, float64(tCell.Y*data.TileSize)-data.TileSize/2)
		x.Image.DrawImage(tile.Image, &opts)
	}

	return nil
}
