package components

import (
	"ebittest/utils"
)

// Do not instantiate directly, use NewVelocityComp().
type Velocity struct {
	vector       utils.Vec2
	acceleration float64
	drag         float64
}

func (Velocity) isComponent() {}
