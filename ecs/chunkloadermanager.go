package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type chunkLoaderManager struct{}

func NewChunkLoaderComponent(radius int) *chunkLoader {
	return &chunkLoader{
		radius: radius,
	}
}

func (chunkLoaderManager) GetRadius(e common.EntityId, ecsContainer *ECSContainer) (int, error) {
	clComp, err := ecsContainer.ChunkLoaders.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get chunk loader component of entity %d: %v", e, err)
	}

	return clComp.radius, nil
}
