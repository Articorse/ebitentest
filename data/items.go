package data

import "math"

const (
	ItemSpriteLayer           = 20
	ItemPickupDelayMs         = 2000
	ItemPickupReachPx         = 3
	ItemTossDistancePx        = 24.0
	ItemTossDistanceVariation = 0.15
	ItemTossAngleVariationRad = 10 * math.Pi / 180
	ItemTossDurationMs        = 300
	ItemTossHeightPx          = 6.0
	ItemPullReachPx           = 25
	ItemPullStartSpeedPx      = 0.25
	ItemPullAccelerationPx    = 0.1
	// How far around where an item's dropped it can end up when there's no room for it there
	GroundSpillRangeTiles = 8
	// How close an item has to be to be picked up on purpose, rather than by walking over it
	ItemPickUpOnPurposeReachPx = 8
)
