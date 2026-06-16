package ecs

import (
	"ebittest/utils"

	"github.com/hajimehoshi/ebiten/v2"
)

type SpriteFlash struct {
	colors           []utils.RelativeColor
	colorDurationsMs []int
	totalDurationMs  int

	colorIdx       int
	counterMs      int
	loopDurationMs int
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
	var flashCopy *SpriteFlash
	if x.flash != nil {
		flashCopy = &SpriteFlash{
			colors:           make([]utils.RelativeColor, len(x.flash.colors)),
			colorDurationsMs: make([]int, len(x.flash.colorDurationsMs)),
			totalDurationMs:  x.flash.totalDurationMs,
			colorIdx:         x.flash.colorIdx,
			counterMs:        x.flash.counterMs,
			loopDurationMs:   x.flash.loopDurationMs,
		}
		copy(flashCopy.colors, x.flash.colors)
		copy(flashCopy.colorDurationsMs, x.flash.colorDurationsMs)
	}

	return sprite{
		image:          x.image,
		offsetPos:      x.offsetPos,
		offsetScale:    x.offsetScale,
		offsetRotation: x.offsetRotation,
		layerYOffset:   x.layerYOffset,
		layer:          x.layer,
		allowRotation:  x.allowRotation,
		flash:          flashCopy,
	}
}
