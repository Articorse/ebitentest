package components

import "ebittest/utils"

// Do not instantiate directly, use NewTransformComp().
type Transform struct {
	Pos      utils.Vec2
	Scale    float64
	Rotation float64
}

func (Transform) isComponent() {}

func NewTransformComponent() *Transform {
	return &Transform{Scale: 1}
}
