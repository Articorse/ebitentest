package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen"

	"github.com/hajimehoshi/ebiten/v2"
)

type chunk struct {
	tiles           [data.ChunkSize * data.ChunkSize]data.TileEnum
	promotedTiles   map[utils.Vec2i]promotedTile
	terrainModified bool

	Image *ebiten.Image
}

func (cc *ChunkContainer) newChunkData(pos utils.Vec2i) (*chunk, error) {
	c := chunk{tiles: worldgen.ChunkTiles(cc.seed, pos)}
	c.promotedTiles = make(map[utils.Vec2i]promotedTile)

	return &c, nil
}

func (x *chunk) getTileDefId(cellKey utils.Vec2i) data.TileEnum {
	localX := ((cellKey.X % data.ChunkSize) + data.ChunkSize) % data.ChunkSize
	localY := ((cellKey.Y % data.ChunkSize) + data.ChunkSize) % data.ChunkSize
	idx := localY*data.ChunkSize + localX
	if len(x.tiles) <= idx {
		return 0
	}

	return x.tiles[idx]
}

func (x *chunk) getPromotedTiles() map[utils.Vec2i]promotedTile {
	return x.promotedTiles
}
