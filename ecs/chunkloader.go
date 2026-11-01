package ecs

import "ebittest/ecs/common"

type chunkLoader struct {
	radius int
}

func (x *chunkLoader) Copy() chunkLoader {
	return chunkLoader{
		radius: x.radius,
	}
}

func (x chunkLoader) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.ChunkLoaders.addComponent(e, x.Copy())
}

type chunkLoaderDto struct {
	Radius int
}

func (x chunkLoader) toDto() (ComponentDto, error) {
	return &chunkLoaderDto{
		Radius: x.radius,
	}, nil
}

func (x chunkLoaderDto) toComponent() (Component, error) {
	return &chunkLoader{
		radius: x.Radius,
	}, nil
}
