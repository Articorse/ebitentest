package ecs

import "ebittest/ecs/collidershapes"

type PlatformColliderManager struct {
	BaseColliderManager[*platformCollider]
}

func NewPlatformColliderComponent(
	shapes []collidershapes.Shape,
) *platformCollider {
	return &platformCollider{baseCollider: newBaseCollider(shapes)}
}

func (x platformCollider) Copy() platformCollider {
	colShapesCopy := make([]collidershapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return platformCollider{
		baseCollider: baseCollider{
			shapes:     colShapesCopy,
			center:     x.center,
			aabb:       x.aabb,
			paddedAabb: x.paddedAabb,
		},
	}
}
