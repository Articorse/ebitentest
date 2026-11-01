package worldgen

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen/noise"
)

// Added to the world seed, so that every layer gets noise of its own
const (
	groundSeed uint64 = iota
	groundFineSeed
	rockCellSeed
	rockChanceSeed
	rockWarpXSeed
	rockWarpYSeed
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
	if isRock(seed, tilePos) {
		return data.Tile_Rock
	}
	if ground(seed, tilePos) > data.WorldgenGrassThreshold {
		return data.Tile_Grass
	}
	return data.Tile_Dirt
}

func ground(seed uint64, tilePos utils.Vec2i) float64 {
	x := float64(tilePos.X) / data.WorldgenGroundScaleTiles
	y := float64(tilePos.Y) / data.WorldgenGroundScaleTiles

	coarse := noise.OpenSimplex2(int64(seed+groundSeed), x, y)
	fine := noise.OpenSimplex2(int64(seed+groundFineSeed), x*2, y*2)
	return coarse + 0.5*fine
}

func isRock(seed uint64, tilePos utils.Vec2i) bool {
	x, y := float64(tilePos.X), float64(tilePos.Y)

	// Warped, so that the borders between cells aren't straight lines
	wx := x + data.WorldgenRockWarpTiles*noise.OpenSimplex2(int64(seed+rockWarpXSeed), x/data.WorldgenRockWarpScaleTiles, y/data.WorldgenRockWarpScaleTiles)
	wy := y + data.WorldgenRockWarpTiles*noise.OpenSimplex2(int64(seed+rockWarpYSeed), x/data.WorldgenRockWarpScaleTiles, y/data.WorldgenRockWarpScaleTiles)

	cellX, cellY := noise.NearestCell(seed+rockCellSeed, wx/data.WorldgenRockCellSizeTiles, wy/data.WorldgenRockCellSizeTiles)
	return noise.UnitFloat(noise.Hash2(seed+rockChanceSeed, cellX, cellY)) < data.WorldgenRockCellChance
}
