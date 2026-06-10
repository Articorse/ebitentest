package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
	"math"
)

type SpawnerManager struct{}

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

func (*SpawnerManager) Spawn(
	spawnerEntity common.EntityId,
	world *World,
) error {
	sm := SpawnerManager{}
	tm := TransformManager{}

	comps, err := sm.GetComponents(spawnerEntity, world.Spawners)
	if err != nil {
		return fmt.Errorf("error getting components to spawn for spawner entity %d: %v", spawnerEntity, err)
	}

	newEntity := world.AddEntity(
		comps...,
	)

	worldPos, _ := tm.GetWorldPos(spawnerEntity, world.Transforms, world.Parents)
	worldRot, _ := tm.GetWorldRotation(spawnerEntity, world.Transforms, world.Parents)

	spawnerOffset, err := sm.GetOffset(spawnerEntity, world.Spawners)
	if err != nil {
		return fmt.Errorf("error getting offset of spawner entity %d: %v", spawnerEntity, err)
	}

	sType, err := sm.GetSpawnerType(spawnerEntity, world.Spawners)
	if err != nil {
		return fmt.Errorf("error getting spawner type of spawner entity %d: %v", spawnerEntity, err)
	}

	shape, err := sm.GetShape(spawnerEntity, world.Spawners)
	if err != nil {
		return fmt.Errorf("error getting shape of spawner entity %d: %v", spawnerEntity, err)
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

	err = tm.SetWorldPos(newEntity, worldPos.Add(finalOffset), world.Transforms, world.Parents)
	if err != nil {
		return fmt.Errorf("error setting world position of new entity %d: %v", newEntity, err)
	}

	err = tm.SetWorldRotation(newEntity, worldRot, world.Transforms, world.Parents)
	if err != nil {
		return fmt.Errorf("error setting world rotation of new entity %d: %v", newEntity, err)
	}

	return nil
}

func (*SpawnerManager) GetOffset(
	e common.EntityId,
	spawners map[common.EntityId]*spawner,
) (utils.Vec2, error) {
	spawnerComp, ok := spawners[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get spawner of entity %d", e)
	}

	return spawnerComp.offset, nil
}

func (*SpawnerManager) GetSpawnerType(
	e common.EntityId,
	spawners map[common.EntityId]*spawner,
) (SpawnerType, error) {
	spawnerComp, ok := spawners[e]
	if !ok {
		return 0, fmt.Errorf("could not get spawner of entity %d", e)
	}

	return spawnerComp.spawnerType, nil
}

func (*SpawnerManager) GetShape(
	e common.EntityId,
	spawners map[common.EntityId]*spawner,
) (shapes.Shape, error) {
	spawnerComp, ok := spawners[e]
	if !ok {
		return nil, fmt.Errorf("could not get spawner of entity %d", e)
	}

	return spawnerComp.shape, nil
}

func (*SpawnerManager) GetComponents(
	e common.EntityId,
	spawners map[common.EntityId]*spawner,
) ([]component, error) {
	spawnerComp, ok := spawners[e]
	if !ok {
		return nil, fmt.Errorf("could not get spawner of entity %d", e)
	}

	return spawnerComp.components, nil
}
