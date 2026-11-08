package worldgen

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen/noise"
	"math"
)

// Added to the world seed, so that every layer gets noise of its own
const (
	groundSeed uint64 = iota
	groundFineSeed
	rockCellSeed
	rockChanceSeed
	rockWarpXSeed
	rockWarpYSeed
	caveSeed
	caveChamberSeed
)

// Every tile depends only on the seed and its own position, so chunks can be generated in any order and regenerated
// at any time
// Layer by layer, as chunks keep them
func ChunkTiles(seed uint64, chunkPos utils.Vec2i) [data.TileLayerCount][data.ChunkSize * data.ChunkSize]data.TileEnum {
	var tiles [data.TileLayerCount][data.ChunkSize * data.ChunkSize]data.TileEnum
	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			layers := Tile(seed, utils.Vec2i{X: chunkPos.X*data.ChunkSize + x, Y: chunkPos.Y*data.ChunkSize + y})
			for l, tile := range layers {
				tiles[l][y*data.ChunkSize+x] = tile
			}
		}
	}
	return tiles
}

// Mountains are roofed whole, rock and caves alike, so that tunnels dug into them stay under their roof
func ChunkRoofs(seed uint64, chunkPos utils.Vec2i) [data.ChunkSize * data.ChunkSize]data.RoofEnum {
	var roofs [data.ChunkSize * data.ChunkSize]data.RoofEnum
	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			roofs[y*data.ChunkSize+x] = Roof(seed, utils.Vec2i{X: chunkPos.X*data.ChunkSize + x, Y: chunkPos.Y*data.ChunkSize + y})
		}
	}
	return roofs
}

func Roof(seed uint64, tilePos utils.Vec2i) data.RoofEnum {
	if isRock(seed, tilePos) {
		return data.Roof_Rock
	}
	return data.Roof_None
}

// Mountains are rock floor walled with rock where there's no cave, and grass grows on dirt
func Tile(seed uint64, tilePos utils.Vec2i) data.TileLayers {
	var tiles data.TileLayers
	if isRock(seed, tilePos) {
		tiles[data.TileLayer_Ground] = data.Tile_RockFloor
		if !isCave(seed, tilePos) {
			tiles[data.TileLayer_Wall] = data.Tile_Rock
		}
		return tiles
	}
	switch water(seed, tilePos) {
	case deep:
		tiles[data.TileLayer_Ground] = data.Tile_WaterDeep
		return tiles
	case shallow:
		tiles[data.TileLayer_Ground] = data.Tile_WaterShallow
		return tiles
	}
	tiles[data.TileLayer_Ground] = data.Tile_Dirt
	if ground(seed, tilePos) > data.WorldgenGrassThreshold {
		tiles[data.TileLayer_Cover] = data.Tile_Grass
	}
	return tiles
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

// Tunnels follow the lines where a noise field crosses zero, which are long and unbroken, so they run through the
// mountains and open onto their edges. A second noise field widens them into chambers, so chambers are always on a
// tunnel.
func isCave(seed uint64, tilePos utils.Vec2i) bool {
	x, y := float64(tilePos.X), float64(tilePos.Y)

	tunnel := noise.OpenSimplex2(int64(seed+caveSeed), x/data.WorldgenCaveScaleTiles, y/data.WorldgenCaveScaleTiles)
	chamber := noise.OpenSimplex2(int64(seed+caveChamberSeed), x/data.WorldgenCaveChamberScaleTiles, y/data.WorldgenCaveChamberScaleTiles)

	width := data.WorldgenCaveTunnelWidth + data.WorldgenCaveChamberWidth*max(chamber-data.WorldgenCaveChamberThreshold, 0)
	return math.Abs(tunnel) < width
}
