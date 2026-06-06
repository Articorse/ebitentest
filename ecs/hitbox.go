package ecs

import (
	"ebittest/ecs/collidershapes"
)

type hitboxCollider struct {
	baseCollider
}

func (hitboxCollider) isComponent() {}

func (x hitboxCollider) Copy() hitboxCollider {
	colShapesCopy := make([]collidershapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return hitboxCollider{
		baseCollider: baseCollider{
			shapes:     colShapesCopy,
			center:     x.center,
			aabb:       x.aabb,
			paddedAabb: x.paddedAabb,
		},
	}
}

func (x *hitboxCollider) getBaseCollider() *baseCollider { return &x.baseCollider }
