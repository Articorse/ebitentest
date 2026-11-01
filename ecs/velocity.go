package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
)

type velocity struct {
	vector       utils.Vec2f
	acceleration float64
	drag         float64
}

func (x velocity) Copy() velocity {
	return velocity{
		vector:       x.vector,
		acceleration: x.acceleration,
		drag:         x.drag,
	}
}

func (x velocity) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Velocities.addComponent(e, x.Copy())
}

type velocityDto struct {
	Vector       utils.Vec2f
	Acceleration float64
	Drag         float64
}

func (x velocity) toDto() (ComponentDto, error) {
	return &velocityDto{
		Vector:       x.vector,
		Acceleration: x.acceleration,
		Drag:         x.drag,
	}, nil
}

func (x *velocityDto) toComponent() (Component, error) {
	return &velocity{
		vector:       x.Vector,
		acceleration: x.Acceleration,
		drag:         x.Drag,
	}, nil
}
