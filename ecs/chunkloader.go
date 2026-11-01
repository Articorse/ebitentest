package ecs

import "ebittest/ecs/common"

type ChunkLoader struct {
	radius int
}

func NewChunkLoaderComponent(radius int) *ChunkLoader {
	return &ChunkLoader{
		radius: radius,
	}
}

func (x *ChunkLoader) Radius() int {
	return x.radius
}

func (x ChunkLoader) Copy() ChunkLoader {
	return ChunkLoader{
		radius: x.radius,
	}
}

func (x ChunkLoader) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.ChunkLoaders.addComponent(e, x.Copy())
}

type chunkLoaderDto struct {
	Radius int
}

func (x ChunkLoader) toDto() (ComponentDto, error) {
	return &chunkLoaderDto{
		Radius: x.radius,
	}, nil
}

func (x chunkLoaderDto) toComponent() (Component, error) {
	return &ChunkLoader{
		radius: x.Radius,
	}, nil
}
