package ecs

import (
	"ebittest/utils"

	"github.com/hajimehoshi/ebiten/v2"
)

type sprite struct {
	image          *ebiten.Image
	offsetPos      utils.Vec2
	offsetScale    float64
	offsetRotation float64
	layerYOffset   uint16
	layer          uint8
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
	}
}
