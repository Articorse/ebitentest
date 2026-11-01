package utils

import (
	"ebittest/data"
	"math"
)

func WorldPosToChunkGridPos(pos Vec2f) Vec2i {
	x := math.Floor(pos.X / (data.ChunkSize * data.TileSize))
	y := math.Floor(pos.Y / (data.ChunkSize * data.TileSize))
	return Vec2i{X: int(x), Y: int(y)}
}

func Dist2(a, b Vec2i) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}

// Tiles are centered on multiples of the tile size
func WorldPosToTilePos(pos Vec2f) Vec2i {
	return Vec2i{
		X: int(math.Floor(pos.X/data.TileSize + 0.5)),
		Y: int(math.Floor(pos.Y/data.TileSize + 0.5)),
	}
}

func TilePosToWorldPos(t Vec2i) Vec2f {
	return Vec2f{X: float64(t.X * data.TileSize), Y: float64(t.Y * data.TileSize)}
}
