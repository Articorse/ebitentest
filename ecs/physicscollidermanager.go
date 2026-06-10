package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

type PhysicsColliderManager struct {
	BaseColliderManager[*physicsCollider]
}

func NewPhysicsColliderComponent(
	cType PhysicsColliderType,
	shapes ...shapes.Shape,
) *physicsCollider {
	return &physicsCollider{
		colliderType: cType,
		baseCollider: newBaseCollider(shapes),
	}
}

func (*PhysicsColliderManager) GetColliderType(
	e common.EntityId,
	world *World,
) (PhysicsColliderType, error) {
	collider, err := world.PhysicsColliders.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return collider.colliderType, nil
}

func (PhysicsColliderManager) EntityIds(w *World) []common.EntityId {
	return w.PhysicsColliders.GetOrderedEntities()
}

func (PhysicsColliderManager) HasCollider(e common.EntityId, w *World) bool {
	return w.PhysicsColliders.HasComponent(e)
}

func (PhysicsColliderManager) GetWorldPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return PhysicsColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, w)
}

func (PhysicsColliderManager) GetShapes(e common.EntityId, w *World) ([]shapes.Shape, error) {
	return PhysicsColliderManager{}.BaseColliderManager.GetShapes(e, w)
}

func (PhysicsColliderManager) GetLocalAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return PhysicsColliderManager{}.BaseColliderManager.GetLocalAABB(e, w)
}

func (PhysicsColliderManager) GetLocalPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return PhysicsColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, w)
}

func (PhysicsColliderManager) GetCenter(e common.EntityId, w *World) (utils.Vec2, error) {
	return PhysicsColliderManager{}.BaseColliderManager.GetCenter(e, w)
}
