package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

// Position, rotation and scale are relative to the parent, if there is one
type Transform struct {
	pos      utils.Vec2f
	scale    float64
	rotation float64
}

func NewTransformComponent(pos utils.Vec2f, scale float64, rotation float64) *Transform {
	return &Transform{pos: pos, scale: scale, rotation: rotation}
}

func (x *Transform) LocalPos() utils.Vec2f {
	return x.pos
}

func (x *Transform) SetLocalPos(pos utils.Vec2f) {
	x.pos = pos
}

func (x *Transform) AddLocalPos(offset utils.Vec2f) {
	x.pos = x.pos.Add(offset)
}

func (x *Transform) LocalRotation() float64 {
	return x.rotation
}

func (x *Transform) SetLocalRotation(rot float64) {
	x.rotation = rot
}

func (x *Transform) AddLocalRotation(rot float64) {
	x.rotation += rot
}

func (x *Transform) LocalScale() float64 {
	return x.scale
}

func (x *Transform) SetLocalScale(scale float64) {
	x.scale = scale
}

func (x *ECSContainer) WorldPos(e common.EntityId) (utils.Vec2f, error) {
	t, ok := x.Transforms.Get(e)
	if !ok {
		return utils.Vec2f{}, fmt.Errorf("entity %d has no transform", e)
	}

	p := x.Parent(e)
	if p == -1 {
		return t.pos, nil
	}

	pWorldPos, err := x.WorldPos(p)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("error getting world position of parent entity %d: %v", p, err)
	}

	pWorldRot, err := x.WorldRotation(p)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("error getting world rotation of parent entity %d: %v", p, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	return utils.Vec2f{
		X: pWorldPos.X + (t.pos.X*cos - t.pos.Y*sin),
		Y: pWorldPos.Y + (t.pos.X*sin + t.pos.Y*cos),
	}, nil
}

func (x *ECSContainer) WorldRotation(e common.EntityId) (float64, error) {
	t, ok := x.Transforms.Get(e)
	if !ok {
		return 0, fmt.Errorf("entity %d has no transform", e)
	}

	p := x.Parent(e)
	if p == -1 {
		return t.rotation, nil
	}

	pWorldRot, err := x.WorldRotation(p)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of parent entity %d: %v", p, err)
	}

	return pWorldRot + t.rotation, nil
}

func (x *ECSContainer) WorldScale(e common.EntityId) (float64, error) {
	t, ok := x.Transforms.Get(e)
	if !ok {
		return 0, fmt.Errorf("entity %d has no transform", e)
	}

	p := x.Parent(e)
	if p == -1 {
		return t.scale, nil
	}

	pWorldScale, err := x.WorldScale(p)
	if err != nil {
		return 0, fmt.Errorf("error getting world scale of parent entity %d: %v", p, err)
	}

	return pWorldScale * t.scale, nil
}

func (x *ECSContainer) SetWorldPos(e common.EntityId, pos utils.Vec2f) error {
	t, ok := x.Transforms.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no transform", e)
	}

	p := x.Parent(e)
	if p == -1 {
		t.pos = pos
		return nil
	}

	pWorldPos, err := x.WorldPos(p)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", p, err)
	}

	pWorldRot, err := x.WorldRotation(p)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", p, err)
	}

	cos := math.Cos(-pWorldRot)
	sin := math.Sin(-pWorldRot)
	relPos := pos.Subtract(pWorldPos)

	t.pos = utils.Vec2f{
		X: relPos.X*cos - relPos.Y*sin,
		Y: relPos.X*sin + relPos.Y*cos,
	}

	return nil
}

func (x *ECSContainer) SetWorldRotation(e common.EntityId, rot float64) error {
	t, ok := x.Transforms.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no transform", e)
	}

	p := x.Parent(e)
	if p == -1 {
		t.rotation = rot
		return nil
	}

	pWorldRot, err := x.WorldRotation(p)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", p, err)
	}

	t.rotation = rot - pWorldRot

	return nil
}

func (x *ECSContainer) SetWorldScale(e common.EntityId, scale float64) error {
	t, ok := x.Transforms.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no transform", e)
	}

	p := x.Parent(e)
	if p == -1 {
		t.scale = scale
		return nil
	}

	pWorldScale, err := x.WorldScale(p)
	if err != nil {
		return fmt.Errorf("error getting world scale of parent entity %d: %v", p, err)
	}

	t.scale = scale / pWorldScale

	return nil
}

func (x Transform) Copy() Transform {
	return Transform{
		pos:      x.pos,
		scale:    x.scale,
		rotation: x.rotation,
	}
}

func (x Transform) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Transforms.addComponent(e, x.Copy())
}

type transformDto struct {
	Pos      utils.Vec2f
	Scale    float64
	Rotation float64
}

func (x Transform) toDto() (ComponentDto, error) {
	return &transformDto{
		Pos:      x.pos,
		Scale:    x.scale,
		Rotation: x.rotation,
	}, nil
}

func (x transformDto) toComponent() (Component, error) {
	return &Transform{
		pos:      x.Pos,
		scale:    x.Scale,
		rotation: x.Rotation,
	}, nil
}
