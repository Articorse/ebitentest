package data

import "image/color"

const (
	VelocityThreshold       = 0.01
	DefaultDrag             = 0.8
	DefaultAcceleration     = 2
	SpatialHashGridCellSize = 200
)

var (
	Debug_ColliderColor             = color.RGBA{0, 255, 0, 255}
	Debug_ColliderCollidedColor     = color.RGBA{255, 0, 0, 255}
	Debug_AABBColliderColor         = color.RGBA{255, 255, 0, 255}
	Debug_AABBColliderCollidedColor = color.RGBA{255, 0, 255, 255}
)
