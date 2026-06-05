package components

import (
	"ebittest/utils"
)

// Do not instantiate directly, use TransformManager.
type Transform struct {
	pos      utils.Vec2
	prevPos  utils.Vec2
	scale    float64
	rotation float64
}

func (Transform) isComponent() {}

func (x Transform) Copy() Transform {
	return Transform{
		pos:      x.pos,
		prevPos:  x.prevPos,
		scale:    x.scale,
		rotation: x.rotation,
	}
}
