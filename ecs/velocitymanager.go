package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

type velocityManager struct{}

func NewDefaultVelocityComponent() *velocity {
	return &velocity{drag: data.DefaultDrag, acceleration: data.DefaultAcceleration}
}

func NewVelocityComponentWithParams(
	vector utils.Vec2,
	acceleration float64,
	drag float64,
) *velocity {
	return &velocity{vector: vector, acceleration: acceleration, drag: drag}
}

func (*velocityManager) GetLocalVector(
	e common.EntityId,
	ecs *ECS,
) (utils.Vec2, error) {
	velComp, err := ecs.Velocities.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get velocity component of entity %d: %v", e, err)
	}

	return velComp.vector, nil
}

func (*velocityManager) GetWorldVector(
	e common.EntityId,
	ecs *ECS,
) (utils.Vec2, error) {
	pm := parentManager{}
	tm := transformManager{}
	vm := velocityManager{}

	velComp, err := ecs.Velocities.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get velocity component of entity %d: %v", e, err)
	}

	parVelVectorOffset := utils.Vec2{}

	parEntity := pm.GetEntity(e, ecs)
	if parEntity != -1 {
		var err error
		parVelVectorOffset, err = vm.GetWorldVector(parEntity, ecs)
		if err != nil {
			return utils.Vec2{}, fmt.Errorf("error getting ecs velocity vector of parent entity %d: %v", parEntity, err)
		}
	}

	ecsRot, err := tm.GetWorldRotation(e, ecs)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting ecs rotation of entity %d: %v", parEntity, err)
	}

	cos := math.Cos(ecsRot)
	sin := math.Sin(ecsRot)

	return utils.Vec2{
		X: parVelVectorOffset.X + (velComp.vector.X*cos - velComp.vector.Y*sin),
		Y: parVelVectorOffset.Y + (velComp.vector.X*sin + velComp.vector.Y*cos),
	}, nil
}

func (*velocityManager) GetAcceleration(
	e common.EntityId,
	ecs *ECS,
) (float64, error) {
	velComp, err := ecs.Velocities.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get velocity component of entity %d: %v", e, err)
	}

	return velComp.acceleration, nil
}

func (*velocityManager) GetDrag(
	e common.EntityId,
	ecs *ECS,
) (float64, error) {
	velComp, err := ecs.Velocities.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get velocity component of entity %d: %v", e, err)
	}

	return velComp.drag, nil
}

func (*velocityManager) AddForce(
	e common.EntityId,
	force utils.Vec2,
	ecs *ECS,
) error {
	valComp, err := ecs.Velocities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get velocity component of entity %d: %v", e, err)
	}

	valComp.vector = valComp.vector.Add(force)
	return nil
}

func (*velocityManager) SetLocalVector(
	e common.EntityId,
	vector utils.Vec2,
	ecs *ECS,
) error {
	valComp, err := ecs.Velocities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get velocity component of entity %d: %v", e, err)
	}

	valComp.vector = vector
	return nil
}

func (*velocityManager) SetWorldVector(
	e common.EntityId,
	vector utils.Vec2,
	ecs *ECS,
) error {
	pm := parentManager{}
	tm := transformManager{}
	vm := velocityManager{}

	velComp, err := ecs.Velocities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get velocity component of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, ecs)
	if parEntity == -1 {
		velComp.vector = vector
		return nil
	}

	pWorldVector, err := vm.GetWorldVector(parEntity, ecs)
	if err != nil {
		return fmt.Errorf("error getting ecs velocity vector of parent entity %d: %v", parEntity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, ecs)
	if err != nil {
		return fmt.Errorf("error getting ecs rotation of parent entity %d: %v", parEntity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	velComp.vector = utils.Vec2{
		X: (velComp.vector.X*cos - velComp.vector.Y*sin) - pWorldVector.X,
		Y: (velComp.vector.X*sin + velComp.vector.Y*cos) - pWorldVector.Y,
	}

	return nil
}

func (*velocityManager) SetDrag(
	e common.EntityId,
	drag float64,
	ecs *ECS,
) error {
	valComp, err := ecs.Velocities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get velocity component of entity %d: %v", e, err)
	}

	valComp.drag = drag
	return nil
}

func (*velocityManager) SetAcceleration(
	e common.EntityId,
	acceleration float64,
	ecs *ECS,
) error {
	valComp, err := ecs.Velocities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get velocity component of entity %d: %v", e, err)
	}

	valComp.acceleration = acceleration
	return nil
}
