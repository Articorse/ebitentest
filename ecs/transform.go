package ecs

import (
	"ebittest/utils"
)

type transform struct {
	pos      utils.Vec2
	scale    float64
	rotation float64
}

func (transform) isComponent() {}

func (x transform) Copy() transform {
	return transform{
		pos:      x.pos,
		scale:    x.scale,
		rotation: x.rotation,
	}
}

type transformDto struct {
	Pos      utils.Vec2
	Scale    float64
	Rotation float64
}

func (transformDto) isComponentDto() {}

func (x transform) ToDto() transformDto {
	return transformDto{
		Pos:      x.pos,
		Scale:    x.scale,
		Rotation: x.rotation,
	}
}

func (x transformDto) ToComponent() *transform {
	return &transform{
		pos:      x.Pos,
		scale:    x.Scale,
		rotation: x.Rotation,
	}
}
