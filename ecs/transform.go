package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
)

type transform struct {
	pos      utils.Vec2f
	scale    float64
	rotation float64
}

func (x transform) Copy() transform {
	return transform{
		pos:      x.pos,
		scale:    x.scale,
		rotation: x.rotation,
	}
}

func (x transform) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Transforms.addComponent(e, x.Copy())
}

type transformDto struct {
	Pos      utils.Vec2f
	Scale    float64
	Rotation float64
}

func (x transform) toDto() (ComponentDto, error) {
	return &transformDto{
		Pos:      x.pos,
		Scale:    x.scale,
		Rotation: x.rotation,
	}, nil
}

func (x transformDto) toComponent() (Component, error) {
	return &transform{
		pos:      x.Pos,
		scale:    x.Scale,
		rotation: x.Rotation,
	}, nil
}
