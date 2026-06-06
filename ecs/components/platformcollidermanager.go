package components

import "ebittest/ecs/components/collidershapes"

type PlatformColliderManager struct {
	BaseColliderManager[*PlatformCollider]
}

func NewPlatformColliderComponent(
	shapes []collidershapes.Shape,
) *PlatformCollider {
	return &PlatformCollider{BaseCollider: newBaseCollider(shapes)}
}

func (x PlatformCollider) Copy() PlatformCollider {
	colShapesCopy := make([]collidershapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return PlatformCollider{
		BaseCollider: BaseCollider{
			shapes:     colShapesCopy,
			center:     x.center,
			aabb:       x.aabb,
			paddedAabb: x.paddedAabb,
		},
	}
}
