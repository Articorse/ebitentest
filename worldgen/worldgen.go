package worldgen

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen/noise"
)

// Every tile depends only on the seed and its own position, so chunks can be generated in any order and regenerated
// at any time
func ChunkTiles(seed uint64, chunkPos utils.Vec2i) [data.ChunkSize * data.ChunkSize]data.TileEnum {
	var tiles [data.ChunkSize * data.ChunkSize]data.TileEnum
	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			tiles[y*data.ChunkSize+x] = Tile(seed, utils.Vec2i{X: chunkPos.X*data.ChunkSize + x, Y: chunkPos.Y*data.ChunkSize + y})
		}
	}
	return tiles
}

func Tile(seed uint64, tilePos utils.Vec2i) data.TileEnum {
	if ground(seed, tilePos) > data.WorldgenGrassThreshold {
		return data.Tile_Grass
	}
	return data.Tile_Dirt
}

func ground(seed uint64, tilePos utils.Vec2i) float64 {
	x := float64(tilePos.X) / data.WorldgenGroundScaleTiles
	y := float64(tilePos.Y) / data.WorldgenGroundScaleTiles

	// Each octave gets its own seed, so the finer one doesn't line up with the coarser one
	coarse := noise.OpenSimplex2(int64(seed), x, y)
	fine := noise.OpenSimplex2(int64(seed+1), x*2, y*2)
	return coarse + 0.5*fine
}
