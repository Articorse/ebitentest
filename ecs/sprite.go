package ecs

import (
	"ebittest/utils"

	"github.com/hajimehoshi/ebiten/v2"
)

type SpriteFlash struct {
	colors           []utils.RelativeColor
	colorDurationsMs []uint64
	totalDurationMs  uint64

	colorIdx       int
	counterMs      uint64
	loopDurationMs uint64
}

type sprite struct {
	image          *ebiten.Image
	offsetPos      utils.Vec2
	offsetScale    float64
	offsetRotation float64
	layerYOffset   uint16
	layer          uint8
	allowRotation  bool
	flash          *SpriteFlash
}

func (sprite) isComponent() {}

func (x sprite) Copy() sprite {
	return sprite{
		image:          x.image,
		offsetPos:      x.offsetPos,
		offsetScale:    x.offsetScale,
		offsetRotation: x.offsetRotation,
		layerYOffset:   x.layerYOffset,
		layer:          x.layer,
		allowRotation:  x.allowRotation,
		flash:          x.flash,
	}
}
