package ecs

import "ebittest/ecs/shapes"

type platformCollider struct {
	baseCollider
}

func (x *platformCollider) getBaseCollider() *baseCollider { return &x.baseCollider }

func (platformCollider) isComponent() {}

func (x platformCollider) Copy() platformCollider {
	colShapesCopy := make([]shapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return platformCollider{
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
