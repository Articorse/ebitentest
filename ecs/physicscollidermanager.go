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

func (physicsColliderManager) GetWorldPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return physicsColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, w)
}

func (physicsColliderManager) GetShapes(e common.EntityId, w *World) ([]shapes.Shape, error) {
	return physicsColliderManager{}.BaseColliderManager.GetShapes(e, w)
}

func (physicsColliderManager) GetLocalAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return physicsColliderManager{}.BaseColliderManager.GetLocalAABB(e, w)
}

func (physicsColliderManager) GetLocalPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return physicsColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, w)
}

func (physicsColliderManager) GetCenter(e common.EntityId, w *World) (utils.Vec2, error) {
	return physicsColliderManager{}.BaseColliderManager.GetCenter(e, w)
}

func (physicsColliderManager) GetLayer(e common.EntityId, world *World) (LayerMask, error) {
	return physicsColliderManager{}.BaseColliderManager.GetLayer(e, world)
}

func (physicsColliderManager) GetMask(e common.EntityId, world *World) (LayerMask, error) {
	return physicsColliderManager{}.BaseColliderManager.GetMask(e, world)
}
