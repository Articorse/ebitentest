package components

import (
	"ebittest/ecs/components/collidershapes"
	"ebittest/ecs/ecscommon"
	"fmt"
)

type PhysicsColliderManager struct {
	BaseColliderManager[*PhysicsCollider]
}

func NewPhysicsColliderComponent(
	cType PhysicsColliderType,
	shapes []collidershapes.Shape,
) *PhysicsCollider {
	return &PhysicsCollider{
		colliderType: cType,
		BaseCollider: newBaseCollider(shapes),
	}
}

func (*PhysicsColliderManager) GetColliderType(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*PhysicsCollider,
) (PhysicsColliderType, error) {
	collider, ok := colliders[e]
	if !ok {
		return 0, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.colliderType, nil
}
