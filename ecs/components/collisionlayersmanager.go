package components

import (
	"ebittest/ecs/ecscommon"
	"fmt"
)

type CollisionLayersManager struct{}

func NewCollisionLayersComponent(
	layers LayerMask,
	mask LayerMask,
) *CollisionLayer {
	return &CollisionLayer{
		layers: layers,
		mask:   mask,
	}
}

func (CollisionLayersManager) GetLayers(
	e ecscommon.EntityId,
	collisionLayers map[ecscommon.EntityId]*CollisionLayer,
) (LayerMask, error) {
	layer, ok := collisionLayers[e]
	if !ok {
		return 0, fmt.Errorf("could not get collider of entity %d", e)
	}

	return layer.layers, nil
}

func (CollisionLayersManager) GetMask(
	e ecscommon.EntityId,
	collisionLayers map[ecscommon.EntityId]*CollisionLayer,
) (LayerMask, error) {
	collider, ok := collisionLayers[e]
	if !ok {
		return 0, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.mask, nil
}
