package ecs

import (
	"ebittest/ecs/collidershapes"
)

type PhysicsColliderType uint8

const (
	Collider_Mob PhysicsColliderType = iota
	Collider_Static
	Collider_Trigger
)

type physicsCollider struct {
	baseCollider

	colliderType PhysicsColliderType
}

func (physicsCollider) isComponent() {}

func (x physicsCollider) Copy() physicsCollider {
	colShapesCopy := make([]collidershapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return physicsCollider{
		colliderType: x.colliderType,
		baseCollider: baseCollider{
			shapes:     colShapesCopy,
			center:     x.center,
			aabb:       x.aabb,
			paddedAabb: x.paddedAabb,
		},
	}
}

func (x *physicsCollider) getBaseCollider() *baseCollider { return &x.baseCollider }
