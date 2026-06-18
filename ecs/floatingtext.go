package ecs

import (
	"ebittest/utils"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type floatingText struct {
	text   string
	offset utils.Vec2
	size   float64
	color  color.RGBA

	face text.GoTextFace
}

func (floatingText) isComponent() {}

func (x floatingText) Copy() floatingText {
	return floatingText{
		text:   x.text,
		offset: x.offset,
		size:   x.size,
		color:  x.color,

		face: x.face,
	}
}
