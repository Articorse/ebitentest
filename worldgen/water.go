package worldgen

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen/noise"
	"math"
)

const (
	lakeCellSeed uint64 = iota + 200
	lakeShapeSeed
	shoreSeed
	pondCellSeed
	pondShapeSeed
)

type waterDepth int

const (
	dry waterDepth = iota
	shallow
	deep
)

func IsWater(tile data.TileEnum) bool {
	return tile == data.Tile_WaterShallow || tile == data.Tile_WaterDeep
}

// Deep water is only kept where every tile around it is water too, so that it's always surrounded by shallow water,
// including where a mountain cuts a lake off
func water(seed uint64, t utils.Vec2i) waterDepth {
	depth := rawWater(seed, t)
	if depth != deep {
		return depth
	}
	for _, n := range neighbors(t) {
		if rawWater(seed, n) == dry || isRock(seed, n) {
			return shallow
		}
	}
	return deep
}

func rawWater(seed uint64, t utils.Vec2i) waterDepth {
	depth := dry
	x, y := float64(t.X), float64(t.Y)

	// Only worked out near a lake, since most tiles aren't
	shore := math.NaN()
	forEachBlob(seed+lakeCellSeed, seed+lakeShapeSeed, x, y, data.WorldgenLakeCellSizeTiles, data.WorldgenLakeChance,
		data.WorldgenLakeMinRadius, data.WorldgenLakeMaxRadius, data.WorldgenShoreMaxWidth, func(d, edge float64) {
			if d < edge {
				depth = deep
				return
			}
			if math.IsNaN(shore) {
				shoreNoise := noise.OpenSimplex2(int64(seed+shoreSeed), x/data.WorldgenShoreScaleTiles, y/data.WorldgenShoreScaleTiles)
				shore = data.WorldgenShoreMinWidth + (data.WorldgenShoreMaxWidth-data.WorldgenShoreMinWidth)*(shoreNoise+1)/2
			}
			if d < edge+shore {
				depth = max(depth, shallow)
			}
		})
	if depth == deep {
		return deep
	}

	forEachBlob(seed+pondCellSeed, seed+pondShapeSeed, x, y, data.WorldgenPondCellSizeTiles, data.WorldgenPondChance,
		data.WorldgenPondMinRadius, data.WorldgenPondMaxRadius, 0, func(d, edge float64) {
			if d < edge {
				depth = max(depth, shallow)
			}
		})
	return depth
}

// Calls blob with how far the tile is from the center of each blob, in the cells around the tile's own, that it may be
// within reach of, and how far the blob's edge is in the tile's direction. A cell holds a blob by chance, placed and
// sized by the cell's hash.
func forEachBlob(cellSeed, shapeSeed uint64, x, y, cellSize, chance, minRadius, maxRadius, reach float64, blob func(d, edge float64)) {
	baseX, baseY := int64(math.Floor(x/cellSize)), int64(math.Floor(y/cellSize))
	for cy := baseY - 1; cy <= baseY+1; cy++ {
		for cx := baseX - 1; cx <= baseX+1; cx++ {
			h := noise.Hash2(cellSeed, cx, cy)
			if noise.UnitFloat(h) >= chance {
				continue
			}
			px, py := noise.FeaturePoint(cellSeed, cx, cy)
			dx, dy := x-px*cellSize, y-py*cellSize
			// Ruling out the cells the biggest blob couldn't reach from first, since most are
			if far := maxRadius*(1+data.WorldgenWaterWobble) + reach; dx*dx+dy*dy >= far*far {
				continue
			}
			radius := minRadius * math.Pow(maxRadius/minRadius, noise.UnitFloat(noise.Hash2(h, 3, 0)))
			d := math.Hypot(dx, dy)
			if d >= radius*(1+data.WorldgenWaterWobble)+reach {
				continue
			}

			wobble := noise.OpenSimplex2(int64(shapeSeed+h), dx/(radius*data.WorldgenWaterWobbleScale), dy/(radius*data.WorldgenWaterWobbleScale))
			blob(d, radius*(1+data.WorldgenWaterWobble*wobble))
		}
	}
}
