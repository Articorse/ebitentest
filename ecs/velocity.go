package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

// The vector is in the entity's own frame, rotated with it and relative to its parent's velocity
type Velocity struct {
	vector       utils.Vec2f
	acceleration float64
	drag         float64
}

func NewDefaultVelocityComponent() *Velocity {
	return &Velocity{drag: data.DefaultDrag, acceleration: data.DefaultAcceleration}
}

func NewVelocityComponentWithParams(
	vector utils.Vec2f,
	acceleration float64,
	drag float64,
) *Velocity {
	return &Velocity{vector: vector, acceleration: acceleration, drag: drag}
}

func (x *Velocity) LocalVector() utils.Vec2f {
	return x.vector
}

func (x *Velocity) SetLocalVector(vector utils.Vec2f) {
	x.vector = vector
}

func (x *Velocity) AddForce(force utils.Vec2f) {
	x.vector = x.vector.Add(force)
}

func (x *Velocity) Acceleration() float64 {
	return x.acceleration
}

func (x *Velocity) SetAcceleration(acceleration float64) {
	x.acceleration = acceleration
}

func (x *Velocity) Drag() float64 {
	return x.drag
}

func (x *Velocity) SetDrag(drag float64) {
	x.drag = drag
}

func (x *ECSContainer) WorldVector(e common.EntityId) (utils.Vec2f, error) {
	v, ok := x.Velocities.Get(e)
	if !ok {
		return utils.Vec2f{}, fmt.Errorf("entity %d has no velocity", e)
	}

	pWorldVector := utils.Vec2f{}

	if p := x.Parent(e); p != -1 {
		var err error
		pWorldVector, err = x.WorldVector(p)
		if err != nil {
			return utils.Vec2f{}, fmt.Errorf("error getting world velocity vector of parent entity %d: %v", p, err)
		}
	}

	worldRot, err := x.WorldRotation(e)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	cos := math.Cos(worldRot)
	sin := math.Sin(worldRot)

	return utils.Vec2f{
		X: pWorldVector.X + (v.vector.X*cos - v.vector.Y*sin),
		Y: pWorldVector.Y + (v.vector.X*sin + v.vector.Y*cos),
	}, nil
}

func (x *ECSContainer) SetWorldVector(e common.EntityId, vector utils.Vec2f) error {
	v, ok := x.Velocities.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no velocity", e)
	}

	pWorldVector := utils.Vec2f{}

	if p := x.Parent(e); p != -1 {
		var err error
		pWorldVector, err = x.WorldVector(p)
		if err != nil {
			return fmt.Errorf("error getting world velocity vector of parent entity %d: %v", p, err)
		}
	}

	worldRot, err := x.WorldRotation(e)
	if err != nil {
		return fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	cos := math.Cos(-worldRot)
	sin := math.Sin(-worldRot)
	relVector := vector.Subtract(pWorldVector)

	v.vector = utils.Vec2f{
		X: relVector.X*cos - relVector.Y*sin,
		Y: relVector.X*sin + relVector.Y*cos,
	}

	return nil
}

func (x Velocity) Copy() Velocity {
	return Velocity{
		vector:       x.vector,
		acceleration: x.acceleration,
		drag:         x.drag,
	}
}

func (x Velocity) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Velocities.addComponent(e, x.Copy())
}

type velocityDto struct {
	Vector       utils.Vec2f
	Acceleration float64
	Drag         float64
}

func (x Velocity) toDto() (ComponentDto, error) {
	return &velocityDto{
		Vector:       x.vector,
		Acceleration: x.acceleration,
		Drag:         x.drag,
	}, nil
}

func (x *velocityDto) toComponent() (Component, error) {
	return &Velocity{
		vector:       x.Vector,
		acceleration: x.Acceleration,
		drag:         x.Drag,
	}, nil
}
