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
	world *World,
) (PhysicsColliderType, error) {
	collider, err := world.PhysicsColliders.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return collider.colliderType, nil
}

func (physicsColliderManager) EntityIds(w *World) []common.EntityId {
	return w.PhysicsColliders.GetEntities()
}

func (physicsColliderManager) HasCollider(e common.EntityId, w *World) bool {
	return w.PhysicsColliders.HasComponent(e)
}

func (physicsColliderManager) IsEnabled(e common.EntityId, world *World) (bool, error) {
	return world.PhysicsColliderManager.BaseColliderManager.IsEnabled(e, world)
}

func (physicsColliderManager) SetEnabled(e common.EntityId, enabled bool, world *World) error {
	return world.PhysicsColliderManager.BaseColliderManager.SetEnabled(e, enabled, world)
}

func (physicsColliderManager) GetWorldPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return world.PhysicsColliderManager.BaseColliderManager.GetWorldPaddedAABB(e, world)
}

func (physicsColliderManager) GetShapes(e common.EntityId, world *World) ([]shapes.Shape, error) {
	return world.PhysicsColliderManager.BaseColliderManager.GetShapes(e, world)
}

func (physicsColliderManager) GetLocalAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return world.PhysicsColliderManager.BaseColliderManager.GetLocalAABB(e, world)
}

func (physicsColliderManager) GetLocalPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return world.PhysicsColliderManager.BaseColliderManager.GetLocalPaddedAABB(e, world)
}

func (physicsColliderManager) GetCenter(e common.EntityId, world *World) (utils.Vec2, error) {
	return world.PhysicsColliderManager.BaseColliderManager.GetCenter(e, world)
}

func (physicsColliderManager) GetLayer(e common.EntityId, world *World) (LayerMask, error) {
	return world.PhysicsColliderManager.BaseColliderManager.GetLayer(e, world)
}

func (physicsColliderManager) GetMask(e common.EntityId, world *World) (LayerMask, error) {
	return world.PhysicsColliderManager.BaseColliderManager.GetMask(e, world)
}
