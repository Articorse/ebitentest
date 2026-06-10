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
	world *World,
) (LayerMask, error) {
	layer, err := world.CollisionLayers.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return layer.layers, nil
}

func (CollisionLayersManager) GetMask(
	e common.EntityId,
	world *World,
) (LayerMask, error) {
	collider, err := world.CollisionLayers.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return collider.mask, nil
}
