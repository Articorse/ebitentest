package components

import "ebittest/utils"

// Do not instantiate directly, use NewTransformComp().
type Transform struct {
	pos      utils.Vec2
	prevPos  utils.Vec2
	scale    float64
	rotation float64
}

func (Transform) isComponent() {}

func NewTransformComponent(pos utils.Vec2, scale float64, rotation float64) *Transform {
	return &Transform{pos: pos, prevPos: pos, scale: scale, rotation: rotation}
}

func (x *Transform) GetPos() utils.Vec2 {
	return x.pos
}

func (x *Transform) SetPos(p utils.Vec2) {
	x.pos = p
}

func (x *Transform) GetPrevPos() utils.Vec2 {
	return x.prevPos
}

func (x *Transform) SetPrevPos(p utils.Vec2) {
	x.prevPos = p
}

func (x *Transform) GetScale() float64 {
	return x.scale
}

func (x *Transform) SetScale(s float64) {
	x.scale = s
}

func (x *Transform) GetRotation() float64 {
	return x.rotation
}

func (x *Transform) SetRotation(r float64) {
	x.rotation = r
}
