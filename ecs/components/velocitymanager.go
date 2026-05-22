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
	velComp, ok := velocities[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get velocity component of entity %d", e)
	}

	parComp, ok := parents[e]
	if !ok {
		return velComp.vector, nil
	}

	if parComp.Entity == -1 {
		return velComp.vector, nil
	}

	tm := TransformManager{}
	vm := VelocityManager{}

	pWorldPos, err := vm.GetWorldVector(parComp.Entity, velocities, transforms, parents)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world velocity vector of parent entity %d: %v", parComp.Entity, ok)
	}

	pWorldRot, err := tm.GetWorldRotation(parComp.Entity, transforms, parents)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world rotation of parent entity %d: %v", parComp.Entity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	return utils.Vec2{
		X: pWorldPos.X + (velComp.vector.X*cos - velComp.vector.Y*sin),
		Y: pWorldPos.Y + (velComp.vector.X*sin + velComp.vector.Y*cos),
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
	traComp, ok := velocities[e]
	if !ok {
		return fmt.Errorf("could not get velocity component of entity %d", e)
	}

	traComp.vector = vector
	return nil
}

func (*VelocityManager) SetWorldVector(
	e ecscommon.EntityId,
	vector utils.Vec2,
	velocities map[ecscommon.EntityId]*Velocity,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) error {
	velComp, ok := velocities[e]
	if !ok {
		return fmt.Errorf("could not get velocity component of entity %d", e)
	}

	parComp, ok := parents[e]
	if !ok {
		velComp.vector = vector
		return nil
	}

	if parComp.Entity == -1 {
		velComp.vector = vector
		return nil
	}

	tm := TransformManager{}
	vm := VelocityManager{}

	pWorldPos, err := vm.GetWorldVector(parComp.Entity, velocities, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world velocity vector of parent entity %d: %v", parComp.Entity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parComp.Entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parComp.Entity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	velComp.vector = utils.Vec2{
		X: (velComp.vector.X*cos - velComp.vector.Y*sin) - pWorldPos.X,
		Y: (velComp.vector.X*sin + velComp.vector.Y*cos) - pWorldPos.Y,
	}

	return nil
}
