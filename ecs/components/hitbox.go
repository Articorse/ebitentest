package components

import (
	"ebittest/ecs/components/collidershapes"
)

type HitboxCollider struct {
	BaseColliderComponent
}

func (HitboxCollider) isComponent() {}

func (x HitboxCollider) Copy() HitboxCollider {
	colShapesCopy := make([]collidershapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return HitboxCollider{
		BaseColliderComponent: BaseColliderComponent{
			shapes:     colShapesCopy,
			center:     x.center,
			aabb:       x.aabb,
			paddedAabb: x.paddedAabb,
		},
	}
}

func (x *HitboxCollider) getBaseCollider() *BaseColliderComponent { return &x.BaseColliderComponent }
