package data

import (
	"image/color"
)

const (
	TPS                            = 60
	CameraWidth                    = 640
	CameraHeight                   = 360
	VelocityThreshold              = 0.01
	DefaultDrag                    = 0.8
	DefaultAcceleration            = 1
	SpatialHashGridCellSize        = 200
	Bounciness                     = 0.0
	AABBPadding                    = 20
	TickMs                         = 1000 / TPS
	MaxAbilitySlots                = 16
	MaxEquipmentAbilitySlots       = 2
	GamepadDeadzone                = 0.15
	GamepadAimDeadzone             = 0.5
	TileSize                       = 16
	ChunkSize                      = 4
	ChunkDataDir                   = "chunkdata"
	ChunkDataFilePathTemplate      = "chunkdata/chunk_%d_%d.gob"
	WorldHeaderFilePath            = "chunkdata/world.gob"
	WorldFormatVersion             = 1
	ChunkReloadRetries             = 3
	MaxChunkLoadDispatchPerTick    = 4
	ChunkDeactivateMargin          = 1
	ChunkPrefetchMargin            = 2
	TetherBorderTiles              = 1
	TetherSearchRadiusTiles        = 16
	ChunkImageBakeMarginPx         = 100
	ChunkImageFreeMarginPx         = 300
	MaxChunkImageBakesPerFrame     = 1
	WorldSavePeriodSec             = 60
	WorldgenGroundScaleTiles       = 24.0
	WorldgenGrassThreshold         = 0.0
	WorldgenRockCellSizeTiles      = 12.0
	WorldgenRockCellChance         = 0.25
	WorldgenRockWarpTiles          = 2.0
	WorldgenRockWarpScaleTiles     = 5.0
	WorldgenSpawnSearchRadiusTiles = 64
	PlayerVisionRadiusTiles        = 12
	VisionCircleMarginTiles        = 1.5
	VisionCircleFadeTiles          = 0.5
)

var (
	Debug_ColliderColor             = color.RGBA{0, 255, 0, 255}
	Debug_ColliderCollidedColor     = color.RGBA{255, 0, 0, 255}
	Debug_AABBColliderColor         = color.RGBA{255, 255, 0, 255}
	Debug_AABBColliderCollidedColor = color.RGBA{255, 0, 255, 255}
	Debug_CollisionVectorColor      = color.RGBA{255, 0, 0, 255}
)
