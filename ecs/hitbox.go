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
	copy(colShapesCopy, x.shapes)

	return hitboxCollider{
		baseCollider: baseCollider{
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
