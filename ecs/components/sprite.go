package components

import (
	"ebittest/utils"

	"github.com/hajimehoshi/ebiten/v2"
)

// Do not instantiate directly, use NewSpriteComp().
type Sprite struct {
	Image          *ebiten.Image
	OffsetPos      utils.Vec2
	OffsetScale    float64
	OffsetRotation float64
	LayerYOffset   uint8
	Layer          uint8
}

func NewSpriteComponent() *Sprite {
	return &Sprite{OffsetScale: 1, Layer: 20}
}
