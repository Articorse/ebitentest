package components

import (
	"ebittest/ecs/components/collidershapes"
)

type HitboxCollider struct {
	BaseCollider
}

func (HitboxCollider) isComponent() {}

func (x HitboxCollider) Copy() HitboxCollider {
	colShapesCopy := make([]collidershapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return HitboxCollider{
		BaseCollider: BaseCollider{
			shapes:     colShapesCopy,
			center:     x.center,
			aabb:       x.aabb,
			paddedAabb: x.paddedAabb,
		},
	}
}

func (x *HitboxCollider) getBaseCollider() *BaseCollider { return &x.BaseCollider }
