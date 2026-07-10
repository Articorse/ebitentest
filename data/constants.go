package data

import (
	"image/color"
)

const (
	TPS                              = 60
	CameraWidth                      = 640
	CameraHeight                     = 360
	VelocityThreshold                = 0.01
	DefaultDrag                      = 0.8
	DefaultAcceleration              = 1
	SpatialHashGridCellSize          = 200
	Bounciness                       = 0.0
	AABBPadding                      = 20
	TickMs                           = 1000 / TPS
	MaxAbilitySlots                  = 16
	MaxEquipmentAbilitySlots         = 2
	GamepadDeadzone                  = 0.15
	GamepadAimDeadzone               = 0.5
	TileSize                         = 16
	ChunkSize                        = 8
	ChunkDataFilePathTemplate        = "chunkdata/chunk_%d_%d.gob"
	RngSeed1                  uint64 = 0b1111110001111110011111000111000010001001001111111000101110001110
	RngSeed2                  uint64 = 0b0101100110101000111000010000011001000001011000011001100101101001
)

var (
	Debug_ColliderColor             = color.RGBA{0, 255, 0, 255}
	Debug_ColliderCollidedColor     = color.RGBA{255, 0, 0, 255}
	Debug_AABBColliderColor         = color.RGBA{255, 255, 0, 255}
	Debug_AABBColliderCollidedColor = color.RGBA{255, 0, 255, 255}
	Debug_CollisionVectorColor      = color.RGBA{255, 0, 0, 255}
)
