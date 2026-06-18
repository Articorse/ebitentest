package ecs

import (
	"ebittest/ecs/shapes"
)

type hitboxCollider struct {
	baseCollider
}

func (hitboxCollider) isComponent() {}

func (x hitboxCollider) Copy() hitboxCollider {
	colShapesCopy := make([]shapes.Shape, len(x.shapes))
	for i, shape := range x.shapes {
		colShapesCopy[i] = shape.Copy()
	}

	return hitboxCollider{
		baseCollider: baseCollider{
			enabled:        x.enabled,
			shapes:         colShapesCopy,
			center:         x.center,
			aabb:           x.aabb,
			paddedAabb:     x.paddedAabb,
			collisionLayer: x.collisionLayer,
			collisionMask:  x.collisionMask,
		},
	}
}

func (x *hitboxCollider) getBaseCollider() *baseCollider { return &x.baseCollider }
