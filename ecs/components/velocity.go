package components

import (
	"ebittest/data"
	"ebittest/utils"
)

// Do not instantiate directly, use NewVelocityComp().
type Velocity struct {
	Vector utils.Vec2
	Drag   float64
}

func NewVelocityComponent() *Velocity {
	return &Velocity{Drag: data.DefaultDrag}
}
