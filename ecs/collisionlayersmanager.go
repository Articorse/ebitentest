package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type CollisionLayersManager struct{}

func NewCollisionLayersComponent(
	layers LayerMask,
	mask LayerMask,
) *collisionLayer {
	return &collisionLayer{
		layers: layers,
		mask:   mask,
	}
}

func (CollisionLayersManager) GetLayers(
	e common.EntityId,
	collisionLayers map[common.EntityId]*collisionLayer,
) (LayerMask, error) {
	layer, ok := collisionLayers[e]
	if !ok {
		return 0, fmt.Errorf("could not get collider of entity %d", e)
	}

	return layer.layers, nil
}

func (CollisionLayersManager) GetMask(
	e common.EntityId,
	collisionLayers map[common.EntityId]*collisionLayer,
) (LayerMask, error) {
	collider, ok := collisionLayers[e]
	if !ok {
		return 0, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.mask, nil
}
