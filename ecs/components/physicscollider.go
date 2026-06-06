package components

import (
	"ebittest/ecs/components/collidershapes"
)

type PhysicsColliderType uint8

const (
	Collider_Mob PhysicsColliderType = iota
	Collider_Static
	Collider_Trigger
)

type PhysicsCollider struct {
	BaseCollider

	colliderType PhysicsColliderType
}

func (PhysicsCollider) isComponent() {}

func (x PhysicsCollider) Copy() PhysicsCollider {
	colShapesCopy := make([]collidershapes.Shape, len(x.shapes))
	copy(colShapesCopy, x.shapes)

	return PhysicsCollider{
		colliderType: x.colliderType,
		BaseCollider: BaseCollider{
			shapes:     colShapesCopy,
			center:     x.center,
			aabb:       x.aabb,
			paddedAabb: x.paddedAabb,
		},
	}
}

func (x *PhysicsCollider) getBaseCollider() *BaseCollider { return &x.BaseCollider }
