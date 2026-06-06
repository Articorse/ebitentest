package ecs

import (
	"ebittest/ecs/collidershapes"
)

type hurtboxCollider struct {
	baseCollider
}

func (hurtboxCollider) isComponent() {}

func (x hurtboxCollider) Copy() hurtboxCollider {
	colShapesCopy := make([]collidershapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return hurtboxCollider{
		baseCollider: baseCollider{
			shapes:     colShapesCopy,
			center:     x.center,
			aabb:       x.aabb,
			paddedAabb: x.paddedAabb,
		},
	}
}

func (x *hurtboxCollider) getBaseCollider() *baseCollider { return &x.baseCollider }
