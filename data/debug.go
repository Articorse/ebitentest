package data

import (
	"image/color"
	"time"
)

// How long the slowest frame shown with the FPS is looked for over
const DebugFrameTimeWindow = time.Second

// Colliders are outlined in the color of their kind, brighter while their entity touches something
var (
	Debug_PhysicsColliderColor         = color.RGBA{0, 160, 0, 255}
	Debug_PhysicsColliderTouchingColor = color.RGBA{140, 255, 140, 255}
	Debug_HitboxColliderColor          = color.RGBA{0, 90, 220, 255}
	Debug_HitboxColliderTouchingColor  = color.RGBA{140, 200, 255, 255}
	Debug_HurtboxColliderColor         = color.RGBA{200, 0, 0, 255}
	Debug_HurtboxColliderTouchingColor = color.RGBA{255, 140, 140, 255}
	Debug_FeatureColliderColor         = color.RGBA{220, 110, 0, 255}
	Debug_FeatureColliderTouchingColor = color.RGBA{255, 190, 110, 255}
	Debug_PlatformColliderColor        = color.RGBA{190, 170, 0, 255}
	// Chunk borders are colored by what the chunk is doing
	Debug_ChunkActiveColor   = color.RGBA{0, 200, 0, 255}
	Debug_ChunkUnbakedColor  = color.RGBA{230, 200, 0, 255}
	Debug_ChunkInactiveColor = color.RGBA{0, 120, 230, 255}
	Debug_ChunkLoadingColor  = color.RGBA{230, 0, 200, 255}
	Debug_PathColor          = color.RGBA{255, 255, 0, 255}
)
