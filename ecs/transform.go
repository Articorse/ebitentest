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
