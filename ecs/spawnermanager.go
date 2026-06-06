package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type SpawnerManager struct{}

func NewSpawnerComponent(offset utils.Vec2, components ...component) *spawner {
	return &spawner{offset: offset, components: components}
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
