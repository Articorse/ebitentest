package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

type chunk struct {
	tiles         [data.ChunkSize * data.ChunkSize]data.TileEnum
	promotedTiles map[utils.Vec2i]promotedTile

	Image *ebiten.Image
}

type chunkMeta struct {
	state        chunkState
	dirty        bool
	saveInFlight bool
	retryAtTick  uint64
}

func (cc *ChunkContainer) newChunkData(pos utils.Vec2i) (*chunk, error) {
	r := rand.NewPCG(data.RngSeed1+uint64(pos.X), data.RngSeed2+uint64(pos.Y))

	c := chunk{}
	c.promotedTiles = make(map[utils.Vec2i]promotedTile)

	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			tileId := data.TileEnum(r.Uint64()%3 + 1)
			c.tiles[y*data.ChunkSize+x] = tileId
		}
	}

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
