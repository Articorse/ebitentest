package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"image/color"
)

type floatingText struct {
	text   string
	offset utils.Vec2f
	size   float64
	color  color.RGBA
}

func (x floatingText) Copy() floatingText {
	return floatingText{
		text:   x.text,
		offset: x.offset,
		size:   x.size,
		color:  x.color,
	}
}

func (x floatingText) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.FloatingTexts.addComponent(e, x.Copy())
}

type floatingTextDto struct {
	Text   string
	Offset utils.Vec2f
	Size   float64
	Color  color.RGBA
}

func (x floatingText) toDto() (ComponentDto, error) {
	return &floatingTextDto{
		Text:   x.text,
		Offset: x.offset,
		Size:   x.size,
		Color:  x.color,
	}, nil
}

func (x floatingTextDto) toComponent() (Component, error) {
	return &floatingText{
		text:   x.Text,
		offset: x.Offset,
		size:   x.Size,
		color:  x.Color,
	}, nil
}
