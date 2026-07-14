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
