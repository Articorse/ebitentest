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
	world *World,
) (common.EntityId, error) {
	sm := spawnerManager{}
	tm := transformManager{}

	comps, err := sm.GetComponents(spawnerEntity, world)
	if err != nil {
		return -1, fmt.Errorf("error getting components to spawn for spawner entity %d: %v", spawnerEntity, err)
	}

	newEntity := world.AddEntity(
		comps...,
	)

	// If there is no Transform, then both of these are zero, which is fine if spawning based on camera or other entity
	worldPos, _ := tm.GetWorldPos(spawnerEntity, world)
	worldRot, _ := tm.GetWorldRotation(spawnerEntity, world)

	spawnerOffset, err := sm.GetOffset(spawnerEntity, world)
	if err != nil {
		world.ScheduleRemoveEntity(newEntity)
		return -1, fmt.Errorf("error getting offset of spawner entity %d: %v", spawnerEntity, err)
	}

	sType, err := sm.GetSpawnerType(spawnerEntity, world)
	if err != nil {
		world.ScheduleRemoveEntity(newEntity)
		return -1, fmt.Errorf("error getting spawner type of spawner entity %d: %v", spawnerEntity, err)
	}

	shape, err := sm.GetShape(spawnerEntity, world)
	if err != nil {
		world.ScheduleRemoveEntity(newEntity)
		return -1, fmt.Errorf("error getting shape of spawner entity %d: %v", spawnerEntity, err)
	}

	var finalOffset utils.Vec2
	switch sType {
	case SpawnerType_Point:
		cos := math.Cos(worldRot)
		sin := math.Sin(worldRot)

		rotatedOffset := utils.Vec2{
			X: (spawnerOffset.X*cos - spawnerOffset.Y*sin),
			Y: (spawnerOffset.X*sin + spawnerOffset.Y*cos),
		}

		finalOffset = rotatedOffset
	case SpawnerType_Inside:
		finalOffset = shape.GetRandomPoint(world.Rng).Add(worldPos).Add(spawnerOffset)
	case SpawnerType_Perimeter:
		finalOffset = shape.GetRandomPointAroundShape(world.Rng).Add(worldPos).Add(spawnerOffset)
	}

	err = tm.SetWorldPos(newEntity, worldPos.Add(finalOffset), world)
	if err != nil {
		world.ScheduleRemoveEntity(newEntity)
		return -1, fmt.Errorf("error setting world position of new entity %d: %v", newEntity, err)
	}

	err = tm.SetWorldRotation(newEntity, worldRot, world)
	if err != nil {
		world.ScheduleRemoveEntity(newEntity)
		return -1, fmt.Errorf("error setting world rotation of new entity %d: %v", newEntity, err)
	}

	return newEntity, nil
}

func (*spawnerManager) GetOffset(
	e common.EntityId,
	world *World,
) (utils.Vec2, error) {
	spawnerComp, err := world.Spawners.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get spawner of entity %d: %v", e, err)
	}

	return spawnerComp.offset, nil
}

func (*spawnerManager) GetSpawnerType(
	e common.EntityId,
	world *World,
) (SpawnerType, error) {
	spawnerComp, err := world.Spawners.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get spawner of entity %d: %v", e, err)
	}

	return spawnerComp.spawnerType, nil
}

func (*spawnerManager) GetShape(
	e common.EntityId,
	world *World,
) (shapes.Shape, error) {
	spawnerComp, err := world.Spawners.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get spawner of entity %d: %v", e, err)
	}

	return spawnerComp.shape, nil
}

func (*spawnerManager) GetComponents(
	e common.EntityId,
	world *World,
) ([]component, error) {
	spawnerComp, err := world.Spawners.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get spawner of entity %d: %v", e, err)
	}

	return spawnerComp.components, nil
}
