package components

import (
	"ebittest/utils"

	"github.com/hajimehoshi/ebiten/v2"
)

// Do not instantiate directly, use NewSpriteComp().
type Sprite struct {
	image          *ebiten.Image
	offsetPos      utils.Vec2
	offsetScale    float64
	offsetRotation float64
	layerYOffset   uint16
	layer          uint8
}

func (Sprite) isComponent() {}

func (x Sprite) Copy() Sprite {
	return Sprite{
		image:          x.image,
		offsetPos:      x.offsetPos,
		offsetScale:    x.offsetScale,
		offsetRotation: x.offsetRotation,
		layerYOffset:   x.layerYOffset,
		layer:          x.layer,
	}
}
