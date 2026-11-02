package utils

import (
	"ebittest/data"
	"math"
)

// Every tile the segment passes through, in order from a. Where it crosses exactly through a corner, the tile beside
// it in x comes before the one in y, so that the same segment always gives the same tiles.
func TilesAlongSegment(a, b Vec2f) []Vec2i {
	start, end := WorldPosToTilePos(a), WorldPosToTilePos(b)
	tiles := []Vec2i{start}

	// In tile units, with tile edges on whole numbers
	ax, ay := a.X/data.TileSize+0.5, a.Y/data.TileSize+0.5
	dx, dy := b.X/data.TileSize+0.5-ax, b.Y/data.TileSize+0.5-ay

	stepX, nextX, deltaX := segmentAxis(ax, dx)
	stepY, nextY, deltaY := segmentAxis(ay, dy)

	t := start
	for t != end {
		if nextX <= nextY {
			t.X += stepX
			nextX += deltaX
		} else {
			t.Y += stepY
			nextY += deltaY
		}
		tiles = append(tiles, t)
	}

	return tiles
}

// The direction to step in, and how far along the segment, as a share of it, the first and every further edge is
func segmentAxis(from, length float64) (step int, next, delta float64) {
	switch {
	case length > 0:
		return 1, (math.Floor(from) + 1 - from) / length, 1 / length
	case length < 0:
		return -1, (from - math.Floor(from)) / -length, 1 / -length
	default:
		return 0, math.Inf(1), math.Inf(1)
	}
}
