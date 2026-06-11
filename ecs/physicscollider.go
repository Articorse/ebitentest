package ecs

import (
	"ebittest/ecs/shapes"
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
	colShapesCopy := make([]shapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return physicsCollider{
		colliderType: x.colliderType,
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

func (x *physicsCollider) getBaseCollider() *baseCollider { return &x.baseCollider }
