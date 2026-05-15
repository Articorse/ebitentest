package components

import (
	"ebittest/utils"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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

func (Sprite) isComponent() {}

func NewSpriteComponent(imageUri string) (*Sprite, error) {
	s := &Sprite{OffsetScale: 1, Layer: 20}
	spr, img, err := ebitenutil.NewImageFromFile(imageUri)
	if err != nil {
		return nil, fmt.Errorf("failed to load sprite image: %w", err)
	}

	s.Image = spr
	s.LayerYOffset = utils.GetFirstOpaquePixelY(img)
	return s, nil
}
