package noise

import "math"

// Each coordinate is mixed in separately, so that swapped or negated coordinates don't hash alike
func Hash2(seed uint64, x, y int64) uint64 {
	return mix(mix(seed^uint64(x)) ^ uint64(y))
}

// Uniform in [0, 1)
func UnitFloat(hash uint64) float64 {
	return float64(hash>>11) / (1 << 53)
}

// The cell, one unit across, whose randomly placed feature point is closest
func NearestCell(seed uint64, x, y float64) (cellX, cellY int64) {
	baseX, baseY := int64(math.Floor(x)), int64(math.Floor(y))
	best := math.Inf(1)

	// Two cells out rather than one, because the own cell's point can be up to √2 away, further than a point two cells
	// over can be
	for dy := int64(-2); dy <= 2; dy++ {
		for dx := int64(-2); dx <= 2; dx++ {
			cx, cy := baseX+dx, baseY+dy
			px, py := FeaturePoint(seed, cx, cy)
			if d := (px-x)*(px-x) + (py-y)*(py-y); d < best {
				best, cellX, cellY = d, cx, cy
			}
		}
	}

	return cellX, cellY
}

func FeaturePoint(seed uint64, cellX, cellY int64) (x, y float64) {
	h := Hash2(seed, cellX, cellY)
	return float64(cellX) + UnitFloat(mix(h+1)), float64(cellY) + UnitFloat(mix(h+2))
}

func mix(v uint64) uint64 {
	v ^= v >> 33
	v *= 0xFF51AFD7ED558CCD
	v ^= v >> 33
	v *= 0xC4CEB9FE1A85EC53
	v ^= v >> 33
	return v
}
