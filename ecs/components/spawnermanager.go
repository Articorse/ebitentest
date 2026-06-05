package components

import (
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
)

type SpawnerManager struct{}

func NewSpawnerComponent(offset utils.Vec2, components []Component) *Spawner {
	return &Spawner{offset: offset, components: components}
}

func (*SpawnerManager) GetOffset(
	e ecscommon.EntityId,
	spawners map[ecscommon.EntityId]*Spawner,
) (utils.Vec2, error) {
	spawnerComp, ok := spawners[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get spawner of entity %d", e)
	}

	return spawnerComp.offset, nil
}

func (*SpawnerManager) GetComponents(
	e ecscommon.EntityId,
	spawners map[ecscommon.EntityId]*Spawner,
) ([]Component, error) {
	spawnerComp, ok := spawners[e]
	if !ok {
		return nil, fmt.Errorf("could not get spawner of entity %d", e)
	}

	return spawnerComp.components, nil
}
