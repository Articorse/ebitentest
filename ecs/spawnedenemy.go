package ecs

import "ebittest/ecs/common"

// Marks an enemy the enemy spawner made, which counts towards its cap and is removed once it's too far away
type SpawnedEnemy struct{}

func NewSpawnedEnemyComponent() *SpawnedEnemy {
	return &SpawnedEnemy{}
}

func (x SpawnedEnemy) Copy() SpawnedEnemy {
	return SpawnedEnemy{}
}

func (x SpawnedEnemy) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.SpawnedEnemies.addComponent(e, x.Copy())
}

type spawnedEnemyDto struct{}

func (x SpawnedEnemy) toDto() (ComponentDto, error) {
	return &spawnedEnemyDto{}, nil
}

func (x spawnedEnemyDto) toComponent() (Component, error) {
	return NewSpawnedEnemyComponent(), nil
}
