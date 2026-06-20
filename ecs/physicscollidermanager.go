package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

type physicsColliderManager struct {
	BaseColliderManager[*physicsCollider]
}

func NewPhysicsColliderComponent(
	collisionLayer LayerMask,
	collisionMask LayerMask,
	cType PhysicsColliderType,
	shapes ...shapes.Shape,
) *physicsCollider {
	return &physicsCollider{
		colliderType: cType,
		baseCollider: newBaseCollider(shapes, collisionLayer, collisionMask),
	}
}

func (*physicsColliderManager) GetColliderType(
	e common.EntityId,
	ecs *ECS,
) (PhysicsColliderType, error) {
	collider, err := ecs.PhysicsColliders.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return collider.colliderType, nil
}

func (physicsColliderManager) EntityIds(w *ECS) []common.EntityId {
	return w.PhysicsColliders.GetEntities()
}

func (physicsColliderManager) HasCollider(e common.EntityId, w *ECS) bool {
	return w.PhysicsColliders.HasComponent(e)
}

func (physicsColliderManager) IsEnabled(e common.EntityId, ecs *ECS) (bool, error) {
	return ecs.PhysicsColliderManager.BaseColliderManager.IsEnabled(e, ecs)
}

func (physicsColliderManager) SetEnabled(e common.EntityId, enabled bool, ecs *ECS) error {
	return ecs.PhysicsColliderManager.BaseColliderManager.SetEnabled(e, enabled, ecs)
}

func (physicsColliderManager) GetWorldPaddedAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return ecs.PhysicsColliderManager.BaseColliderManager.GetWorldPaddedAABB(e, ecs)
}

func (physicsColliderManager) GetShapes(e common.EntityId, ecs *ECS) ([]shapes.Shape, error) {
	return ecs.PhysicsColliderManager.BaseColliderManager.GetShapes(e, ecs)
}

func (physicsColliderManager) GetLocalAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return ecs.PhysicsColliderManager.BaseColliderManager.GetLocalAABB(e, ecs)
}

func (physicsColliderManager) GetLocalPaddedAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return ecs.PhysicsColliderManager.BaseColliderManager.GetLocalPaddedAABB(e, ecs)
}

func (physicsColliderManager) GetCenter(e common.EntityId, ecs *ECS) (utils.Vec2, error) {
	return ecs.PhysicsColliderManager.BaseColliderManager.GetCenter(e, ecs)
}

func (physicsColliderManager) GetLayer(e common.EntityId, ecs *ECS) (LayerMask, error) {
	return ecs.PhysicsColliderManager.BaseColliderManager.GetLayer(e, ecs)
}

func (physicsColliderManager) GetMask(e common.EntityId, ecs *ECS) (LayerMask, error) {
	return ecs.PhysicsColliderManager.BaseColliderManager.GetMask(e, ecs)
}
