package components

import (
	"ebittest/data"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
	"math"
)

type VelocityManager struct{}

func NewVelocityComponent() *Velocity {
	return &Velocity{drag: data.DefaultDrag, acceleration: data.DefaultAcceleration}
}

func (*VelocityManager) GetLocalVector(
	e ecscommon.EntityId,
	velocities map[ecscommon.EntityId]*Velocity,
) (utils.Vec2, error) {
	velComp, ok := velocities[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get velocity component of entity %d", e)
	}

	return velComp.vector, nil
}

func (*VelocityManager) GetWorldVector(
	e ecscommon.EntityId,
	velocities map[ecscommon.EntityId]*Velocity,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) (utils.Vec2, error) {
	pm := ParentManager{}
	tm := TransformManager{}
	vm := VelocityManager{}

	velComp, ok := velocities[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get velocity component of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		return velComp.vector, nil
	}

	pWorldVelVec, err := vm.GetWorldVector(parEntity, velocities, transforms, parents)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world velocity vector of parent entity %d: %v", parEntity, ok)
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, transforms, parents)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	return utils.Vec2{
		X: pWorldVelVec.X + (velComp.vector.X*cos - velComp.vector.Y*sin),
		Y: pWorldVelVec.Y + (velComp.vector.X*sin + velComp.vector.Y*cos),
	}, nil
}

func (*VelocityManager) GetAcceleration(
	e ecscommon.EntityId,
	velocities map[ecscommon.EntityId]*Velocity,
) (float64, error) {
	velComp, ok := velocities[e]
	if !ok {
		return 0, fmt.Errorf("could not get velocity component of entity %d", e)
	}

	return velComp.acceleration, nil
}

func (*VelocityManager) GetDrag(
	e ecscommon.EntityId,
	velocities map[ecscommon.EntityId]*Velocity,
) (float64, error) {
	velComp, ok := velocities[e]
	if !ok {
		return 0, fmt.Errorf("could not get velocity component of entity %d", e)
	}

	return velComp.drag, nil
}

func (*VelocityManager) SetLocalVector(
	e ecscommon.EntityId,
	vector utils.Vec2,
	velocities map[ecscommon.EntityId]*Velocity,
) error {
	valComp, ok := velocities[e]
	if !ok {
		return fmt.Errorf("could not get velocity component of entity %d", e)
	}

	valComp.vector = vector
	return nil
}

func (*VelocityManager) SetWorldVector(
	e ecscommon.EntityId,
	vector utils.Vec2,
	velocities map[ecscommon.EntityId]*Velocity,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
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
	e ecscommon.EntityId,
	drag float64,
	velocities map[ecscommon.EntityId]*Velocity,
) error {
	valComp, ok := velocities[e]
	if !ok {
		return fmt.Errorf("could not get velocity component of entity %d", e)
	}

	valComp.drag = drag
	return nil
}

func (*VelocityManager) SetAcceleration(
	e ecscommon.EntityId,
	acceleration float64,
	velocities map[ecscommon.EntityId]*Velocity,
) error {
	valComp, ok := velocities[e]
	if !ok {
		return fmt.Errorf("could not get velocity component of entity %d", e)
	}

	valComp.acceleration = acceleration
	return nil
}
