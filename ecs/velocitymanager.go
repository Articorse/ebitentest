package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

type VelocityManager struct{}

func NewDefaultVelocityComponent() *velocity {
	return &velocity{drag: data.DefaultDrag, acceleration: data.DefaultAcceleration}
}

func NewVelocityComponent(
	vector utils.Vec2,
	drag float64,
	acceleration float64,
) *velocity {
	return &velocity{vector: vector, drag: drag, acceleration: acceleration}
}

func NewVelocityComponentWithParams(
	vector utils.Vec2,
	acceleration float64,
	drag float64,
) *velocity {
	return &velocity{vector: vector, acceleration: acceleration, drag: drag}
}

func (*VelocityManager) GetLocalVector(
	e common.EntityId,
	velocities map[common.EntityId]*velocity,
) (utils.Vec2, error) {
	velComp, ok := velocities[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get velocity component of entity %d", e)
	}

	return velComp.vector, nil
}

func (*VelocityManager) GetWorldVector(
	e common.EntityId,
	velocities map[common.EntityId]*velocity,
	transforms map[common.EntityId]*transform,
	parents map[common.EntityId]*parent,
) (utils.Vec2, error) {
	pm := ParentManager{}
	tm := TransformManager{}
	vm := VelocityManager{}

	velComp, ok := velocities[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get velocity component of entity %d", e)
	}

	parVelVectorOffset := utils.Vec2{}

	parEntity := pm.GetEntity(e, parents)
	if parEntity != -1 {
		var err error
		parVelVectorOffset, err = vm.GetWorldVector(parEntity, velocities, transforms, parents)
		if err != nil {
			return utils.Vec2{}, fmt.Errorf("error getting world velocity vector of parent entity %d: %v", parEntity, ok)
		}
	}

	worldRot, err := tm.GetWorldRotation(e, transforms, parents)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world rotation of entity %d: %v", parEntity, err)
	}

	cos := math.Cos(worldRot)
	sin := math.Sin(worldRot)

	return utils.Vec2{
		X: parVelVectorOffset.X + (velComp.vector.X*cos - velComp.vector.Y*sin),
		Y: parVelVectorOffset.Y + (velComp.vector.X*sin + velComp.vector.Y*cos),
	}, nil
}

func (*VelocityManager) GetAcceleration(
	e common.EntityId,
	velocities map[common.EntityId]*velocity,
) (float64, error) {
	velComp, ok := velocities[e]
	if !ok {
		return 0, fmt.Errorf("could not get velocity component of entity %d", e)
	}

	return velComp.acceleration, nil
}

func (*VelocityManager) GetDrag(
	e common.EntityId,
	velocities map[common.EntityId]*velocity,
) (float64, error) {
	velComp, ok := velocities[e]
	if !ok {
		return 0, fmt.Errorf("could not get velocity component of entity %d", e)
	}

	return velComp.drag, nil
}

func (*VelocityManager) AddForce(
	e common.EntityId,
	force utils.Vec2,
	velocities map[common.EntityId]*velocity,
) error {
	valComp, ok := velocities[e]
	if !ok {
		return fmt.Errorf("could not get velocity component of entity %d", e)
	}

	valComp.vector = valComp.vector.Add(force)
	return nil
}

func (*VelocityManager) SetLocalVector(
	e common.EntityId,
	vector utils.Vec2,
	velocities map[common.EntityId]*velocity,
) error {
	valComp, ok := velocities[e]
	if !ok {
		return fmt.Errorf("could not get velocity component of entity %d", e)
	}

	valComp.vector = vector
	return nil
}

func (*VelocityManager) SetWorldVector(
	e common.EntityId,
	vector utils.Vec2,
	velocities map[common.EntityId]*velocity,
	transforms map[common.EntityId]*transform,
	parents map[common.EntityId]*parent,
) error {
	pm := ParentManager{}
	tm := TransformManager{}
	vm := VelocityManager{}

	velComp, ok := velocities[e]
	if !ok {
		return fmt.Errorf("could not get velocity component of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		velComp.vector = vector
		return nil
	}

	pWorldVector, err := vm.GetWorldVector(parEntity, velocities, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world velocity vector of parent entity %d: %v", parEntity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	velComp.vector = utils.Vec2{
		X: (velComp.vector.X*cos - velComp.vector.Y*sin) - pWorldVector.X,
		Y: (velComp.vector.X*sin + velComp.vector.Y*cos) - pWorldVector.Y,
	}

	return nil
}

func (*VelocityManager) SetDrag(
	e common.EntityId,
	drag float64,
	velocities map[common.EntityId]*velocity,
) error {
	valComp, ok := velocities[e]
	if !ok {
		return fmt.Errorf("could not get velocity component of entity %d", e)
	}

	valComp.drag = drag
	return nil
}

func (*VelocityManager) SetAcceleration(
	e common.EntityId,
	acceleration float64,
	velocities map[common.EntityId]*velocity,
) error {
	valComp, ok := velocities[e]
	if !ok {
		return fmt.Errorf("could not get velocity component of entity %d", e)
	}

	valComp.acceleration = acceleration
	return nil
}
