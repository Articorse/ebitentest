package ecs

import (
	"ebittest/utils"
)

type velocity struct {
	vector       utils.Vec2
	acceleration float64
	drag         float64
}

func (velocity) isComponent() {}

func (x velocity) Copy() velocity {
	return velocity{
		vector:       x.vector,
		acceleration: x.acceleration,
		drag:         x.drag,
	}
}

type velocityDto struct {
	Vector       utils.Vec2
	Acceleration float64
	Drag         float64
}

func (velocityDto) isComponentDto() {}

func (x velocity) ToDto() velocityDto {
	return velocityDto{
		Vector:       x.vector,
		Acceleration: x.acceleration,
		Drag:         x.drag,
	}
}

func (x *velocityDto) ToComponent() *velocity {
	return &velocity{
		vector:       x.Vector,
		acceleration: x.Acceleration,
		drag:         x.Drag,
	}
}
