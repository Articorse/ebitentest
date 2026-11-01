package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"image/color"
)

type FloatingText struct {
	text   string
	offset utils.Vec2f
	size   float64
	color  color.RGBA
}

func NewFloatingTextComponent(ptext string, pos utils.Vec2f, size float64, color color.RGBA) *FloatingText {
	return &FloatingText{
		text:   ptext,
		offset: pos,
		size:   size,
		color:  color,
	}
}

func (x *FloatingText) Text() string {
	return x.text
}

func (x *FloatingText) Offset() utils.Vec2f {
	return x.offset
}

func (x *FloatingText) Size() float64 {
	return x.size
}

func (x *FloatingText) Color() color.RGBA {
	return x.color
}

func (x FloatingText) Copy() FloatingText {
	return FloatingText{
		text:   x.text,
		offset: x.offset,
		size:   x.size,
		color:  x.color,
	}
}

func (x FloatingText) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.FloatingTexts.addComponent(e, x.Copy())
}

type floatingTextDto struct {
	Text   string
	Offset utils.Vec2f
	Size   float64
	Color  color.RGBA
}

func (x FloatingText) toDto() (ComponentDto, error) {
	return &floatingTextDto{
		Text:   x.text,
		Offset: x.offset,
		Size:   x.size,
		Color:  x.color,
	}, nil
}

func (x floatingTextDto) toComponent() (Component, error) {
	return &FloatingText{
		text:   x.Text,
		offset: x.Offset,
		size:   x.Size,
		color:  x.Color,
	}, nil
}
