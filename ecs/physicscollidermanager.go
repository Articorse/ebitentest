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
	ecsContainer *ECSContainer,
) (PhysicsColliderType, error) {
	collider, err := ecsContainer.PhysicsColliders.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return collider.colliderType, nil
}

func (physicsColliderManager) EntityIds(w *ECSContainer) []common.EntityId {
	return w.PhysicsColliders.GetEntities()
}

func (physicsColliderManager) HasCollider(e common.EntityId, w *ECSContainer) bool {
	return w.PhysicsColliders.HasComponent(e)
}

func (physicsColliderManager) IsEnabled(e common.EntityId, ecsContainer *ECSContainer) (bool, error) {
	return ecsContainer.PhysicsColliderManager.BaseColliderManager.IsEnabled(e, ecsContainer)
}

func (physicsColliderManager) SetEnabled(e common.EntityId, enabled bool, ecsContainer *ECSContainer) error {
	return ecsContainer.PhysicsColliderManager.BaseColliderManager.SetEnabled(e, enabled, ecsContainer)
}

func (physicsColliderManager) GetecsContainerPaddedAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2, error) {
	return ecsContainer.PhysicsColliderManager.BaseColliderManager.GetWorldPaddedAABB(e, ecsContainer)
}

func (physicsColliderManager) GetShapes(e common.EntityId, ecsContainer *ECSContainer) ([]shapes.Shape, error) {
	return ecsContainer.PhysicsColliderManager.BaseColliderManager.GetShapes(e, ecsContainer)
}

func (physicsColliderManager) GetLocalAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2, error) {
	return ecsContainer.PhysicsColliderManager.BaseColliderManager.GetLocalAABB(e, ecsContainer)
}

func (physicsColliderManager) GetLocalPaddedAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2, error) {
	return ecsContainer.PhysicsColliderManager.BaseColliderManager.GetLocalPaddedAABB(e, ecsContainer)
}

func (physicsColliderManager) GetCenter(e common.EntityId, ecsContainer *ECSContainer) (utils.Vec2, error) {
	return ecsContainer.PhysicsColliderManager.BaseColliderManager.GetCenter(e, ecsContainer)
}

func (physicsColliderManager) GetLayer(e common.EntityId, ecsContainer *ECSContainer) (LayerMask, error) {
	return ecsContainer.PhysicsColliderManager.BaseColliderManager.GetLayer(e, ecsContainer)
}

func (physicsColliderManager) GetMask(e common.EntityId, ecsContainer *ECSContainer) (LayerMask, error) {
	return ecsContainer.PhysicsColliderManager.BaseColliderManager.GetMask(e, ecsContainer)
}
