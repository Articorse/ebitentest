package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
	"math"
)

type spawnerManager struct{}

func NewSpawnerComponent(
	offset utils.Vec2,
	sType SpawnerType,
	shape shapes.Shape,
	components ...component,
) (*spawner, error) {
	switch sType {
	case SpawnerType_Point:
		if shape != nil {
			return nil, fmt.Errorf("shape must be nil for point spawner type")
		}
	case SpawnerType_Inside, SpawnerType_Perimeter:
		if shape == nil {
			return nil, fmt.Errorf("shape must be non-nil for inside and perimeter spawner types")
		}
	default:
		return nil, fmt.Errorf("invalid spawner type: %d", sType)
	}

	return &spawner{offset: offset, spawnerType: sType, shape: shape, components: components}, nil
}

func (*spawnerManager) Spawn(
	spawnerEntity common.EntityId,
	ecs *ECS,
) (common.EntityId, error) {
	sm := spawnerManager{}
	tm := transformManager{}

	comps, err := sm.GetComponents(spawnerEntity, ecs)
	if err != nil {
		return -1, fmt.Errorf("error getting components to spawn for spawner entity %d: %v", spawnerEntity, err)
	}

	newEntity := ecs.AddEntity(
		comps...,
	)

	// If there is no Transform, then both of these are zero, which is fine if spawning based on camera or other entity
	ecsPos, _ := tm.GetWorldPos(spawnerEntity, ecs)
	ecsRot, _ := tm.GetWorldRotation(spawnerEntity, ecs)

	spawnerOffset, err := sm.GetOffset(spawnerEntity, ecs)
	if err != nil {
		ecs.ScheduleRemoveEntity(newEntity)
		return -1, fmt.Errorf("error getting offset of spawner entity %d: %v", spawnerEntity, err)
	}

	sType, err := sm.GetSpawnerType(spawnerEntity, ecs)
	if err != nil {
		ecs.ScheduleRemoveEntity(newEntity)
		return -1, fmt.Errorf("error getting spawner type of spawner entity %d: %v", spawnerEntity, err)
	}

	shape, err := sm.GetShape(spawnerEntity, ecs)
	if err != nil {
		ecs.ScheduleRemoveEntity(newEntity)
		return -1, fmt.Errorf("error getting shape of spawner entity %d: %v", spawnerEntity, err)
	}

	var finalOffset utils.Vec2
	switch sType {
	case SpawnerType_Point:
		cos := math.Cos(ecsRot)
		sin := math.Sin(ecsRot)

		rotatedOffset := utils.Vec2{
			X: (spawnerOffset.X*cos - spawnerOffset.Y*sin),
			Y: (spawnerOffset.X*sin + spawnerOffset.Y*cos),
		}

		finalOffset = rotatedOffset
	case SpawnerType_Inside:
		finalOffset = shape.GetRandomPoint(ecs.Rng).Add(ecsPos).Add(spawnerOffset)
	case SpawnerType_Perimeter:
		finalOffset = shape.GetRandomPointAroundShape(ecs.Rng).Add(ecsPos).Add(spawnerOffset)
	}

	err = tm.SetWorldPos(newEntity, ecsPos.Add(finalOffset), ecs)
	if err != nil {
		ecs.ScheduleRemoveEntity(newEntity)
		return -1, fmt.Errorf("error setting ecs position of new entity %d: %v", newEntity, err)
	}

	err = tm.SetWorldRotation(newEntity, ecsRot, ecs)
	if err != nil {
		ecs.ScheduleRemoveEntity(newEntity)
		return -1, fmt.Errorf("error setting ecs rotation of new entity %d: %v", newEntity, err)
	}

	return newEntity, nil
}

func (*spawnerManager) GetOffset(
	e common.EntityId,
	ecs *ECS,
) (utils.Vec2, error) {
	spawnerComp, err := ecs.Spawners.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get spawner of entity %d: %v", e, err)
	}

	return spawnerComp.offset, nil
}

func (*spawnerManager) GetSpawnerType(
	e common.EntityId,
	ecs *ECS,
) (SpawnerType, error) {
	spawnerComp, err := ecs.Spawners.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get spawner of entity %d: %v", e, err)
	}

	return spawnerComp.spawnerType, nil
}

func (*spawnerManager) GetShape(
	e common.EntityId,
	ecs *ECS,
) (shapes.Shape, error) {
	spawnerComp, err := ecs.Spawners.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get spawner of entity %d: %v", e, err)
	}

	return spawnerComp.shape, nil
}

func (*spawnerManager) GetComponents(
	e common.EntityId,
	ecs *ECS,
) ([]component, error) {
	spawnerComp, err := ecs.Spawners.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get spawner of entity %d: %v", e, err)
	}

	return spawnerComp.components, nil
}
