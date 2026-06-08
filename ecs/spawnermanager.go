package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
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
