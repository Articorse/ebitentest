package ecs

import (
	"ebittest/ecs/shapes"
)

type hurtboxCollider struct {
	baseCollider
}

func (hurtboxCollider) isComponent() {}

func (x hurtboxCollider) Copy() hurtboxCollider {
	colShapesCopy := make([]shapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return hurtboxCollider{
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

func (x *hurtboxCollider) getBaseCollider() *baseCollider { return &x.baseCollider }
