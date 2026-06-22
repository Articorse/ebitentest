package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
)

type hurtboxColliderManager struct {
	BaseColliderManager[*hurtboxCollider]
}

func NewHurtboxColliderComponent(
	collisionLayer LayerMask,
	collisionMask LayerMask,
	shapes ...shapes.Shape,
) *hurtboxCollider {
	return &hurtboxCollider{baseCollider: newBaseCollider(shapes, collisionLayer, collisionMask)}
}

func (hurtboxColliderManager) EntityIds(ecsContainer *ECSContainer) []common.EntityId {
	return ecsContainer.HurtboxColliders.GetEntities()
}

func (hurtboxColliderManager) HasCollider(e common.EntityId, ecsContainer *ECSContainer) bool {
	return ecsContainer.HurtboxColliders.HasComponent(e)
}

func (hurtboxColliderManager) IsEnabled(e common.EntityId, ecsContainer *ECSContainer) (bool, error) {
	return ecsContainer.HurtboxColliderManager.BaseColliderManager.IsEnabled(e, ecsContainer)
}

func (hurtboxColliderManager) SetEnabled(e common.EntityId, enabled bool, ecsContainer *ECSContainer) error {
	return ecsContainer.HurtboxColliderManager.BaseColliderManager.SetEnabled(e, enabled, ecsContainer)
}

func (hurtboxColliderManager) GetecsContainerPaddedAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2, error) {
	return ecsContainer.HurtboxColliderManager.BaseColliderManager.GetWorldPaddedAABB(e, ecsContainer)
}

func (hurtboxColliderManager) GetShapes(e common.EntityId, ecsContainer *ECSContainer) ([]shapes.Shape, error) {
	return ecsContainer.HurtboxColliderManager.BaseColliderManager.GetShapes(e, ecsContainer)
}

func (hurtboxColliderManager) GetLocalAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2, error) {
	return ecsContainer.HurtboxColliderManager.BaseColliderManager.GetLocalAABB(e, ecsContainer)
}

func (hurtboxColliderManager) GetLocalPaddedAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2, error) {
	return ecsContainer.HurtboxColliderManager.BaseColliderManager.GetLocalPaddedAABB(e, ecsContainer)
}

func (hurtboxColliderManager) GetCenter(e common.EntityId, ecsContainer *ECSContainer) (utils.Vec2, error) {
	return ecsContainer.HurtboxColliderManager.BaseColliderManager.GetCenter(e, ecsContainer)
}

func (hurtboxColliderManager) GetLayer(e common.EntityId, ecsContainer *ECSContainer) (LayerMask, error) {
	return ecsContainer.HurtboxColliderManager.BaseColliderManager.GetLayer(e, ecsContainer)
}

func (hurtboxColliderManager) GetMask(e common.EntityId, ecsContainer *ECSContainer) (LayerMask, error) {
	return ecsContainer.HurtboxColliderManager.BaseColliderManager.GetMask(e, ecsContainer)
}
