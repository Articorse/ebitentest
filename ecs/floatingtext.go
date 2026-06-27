package ecs

import (
	"ebittest/utils"
	"image/color"
)

type floatingText struct {
	text   string
	offset utils.Vec2
	size   float64
	color  color.RGBA
}

func (floatingText) isComponent() {}

func (x floatingText) Copy() floatingText {
	return floatingText{
		text:   x.text,
		offset: x.offset,
		size:   x.size,
		color:  x.color,
	}
}

type floatingTextDto struct {
	Text   string
	Offset utils.Vec2
	Size   float64
	Color  color.RGBA
}

func (floatingTextDto) isComponentDto() {}

func (x floatingText) ToDto() floatingTextDto {
	return floatingTextDto{
		Text:   x.text,
		Offset: x.offset,
		Size:   x.size,
		Color:  x.color,
	}
}

func (x floatingTextDto) ToComponent() *floatingText {
	return &floatingText{
		text:   x.Text,
		offset: x.Offset,
		size:   x.Size,
		color:  x.Color,
	}
}
