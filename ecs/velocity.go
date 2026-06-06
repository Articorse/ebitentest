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
