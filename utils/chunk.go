package utils

import (
	"ebittest/data"
	"math"
)

type CellKey struct {
	X int
	Y int
}

func WorldPosToChunkGridPos(pos Vec2) CellKey {
	x := math.Floor(pos.X / (data.ChunkSize * data.TileSize))
	y := math.Floor(pos.Y / (data.ChunkSize * data.TileSize))
	return CellKey{X: int(x), Y: int(y)}
}

func Dist2(a, b CellKey) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}
