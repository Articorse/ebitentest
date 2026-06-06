package components

import (
	"ebittest/ecs/components/collidershapes"
)

type HurtboxCollider struct {
	BaseCollider
}

func (HurtboxCollider) isComponent() {}

func (x HurtboxCollider) Copy() HurtboxCollider {
	colShapesCopy := make([]collidershapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return HurtboxCollider{
		BaseCollider: BaseCollider{
			shapes:     colShapesCopy,
			center:     x.center,
			aabb:       x.aabb,
			paddedAabb: x.paddedAabb,
		},
	}
}

func (x *HurtboxCollider) getBaseCollider() *BaseCollider { return &x.BaseCollider }
