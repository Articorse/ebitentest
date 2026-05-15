package components

import (
	"ebittest/data"
	"ebittest/utils"
)

// Do not instantiate directly, use NewVelocityComp().
type Velocity struct {
	Vector       utils.Vec2
	Acceleration float64
	Drag         float64
}

func (Velocity) isComponent() {}

func NewVelocityComponent() *Velocity {
	return &Velocity{Drag: data.DefaultDrag, Acceleration: data.DefaultAcceleration}
}
