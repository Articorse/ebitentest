package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
)

type hitboxColliderManager struct {
	BaseColliderManager[*hitboxCollider]
}

func NewHitboxColliderComponent(
	collisionLayer LayerMask,
	collisionMask LayerMask,
	shapes ...shapes.Shape,
) *hitboxCollider {
	return &hitboxCollider{baseCollider: newBaseCollider(shapes, collisionLayer, collisionMask)}
}

func (hitboxColliderManager) EntityIds(ecsContainer *ECSContainer) []common.EntityId {
	return ecsContainer.HitboxColliders.GetEntities()
}

func (hitboxColliderManager) HasCollider(e common.EntityId, ecsContainer *ECSContainer) bool {
	return ecsContainer.HitboxColliders.HasComponent(e)
}

func (hitboxColliderManager) IsEnabled(e common.EntityId, ecsContainer *ECSContainer) (bool, error) {
	return ecsContainer.HitboxColliderManager.BaseColliderManager.IsEnabled(e, ecsContainer)
}

func (hitboxColliderManager) SetEnabled(e common.EntityId, enabled bool, ecsContainer *ECSContainer) error {
	return ecsContainer.HitboxColliderManager.BaseColliderManager.SetEnabled(e, enabled, ecsContainer)
}

func (hitboxColliderManager) GetecsContainerPaddedAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2f, error) {
	return ecsContainer.HitboxColliderManager.BaseColliderManager.GetWorldPaddedAABB(e, ecsContainer)
}

func (hitboxColliderManager) GetShapes(e common.EntityId, ecsContainer *ECSContainer) ([]shapes.Shape, error) {
	return ecsContainer.HitboxColliderManager.BaseColliderManager.GetShapes(e, ecsContainer)
}

func (hitboxColliderManager) GetLocalAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2f, error) {
	return ecsContainer.HitboxColliderManager.BaseColliderManager.GetLocalAABB(e, ecsContainer)
}

func (hitboxColliderManager) GetLocalPaddedAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2f, error) {
	return ecsContainer.HitboxColliderManager.BaseColliderManager.GetLocalPaddedAABB(e, ecsContainer)
}

func (hitboxColliderManager) GetCenter(e common.EntityId, ecsContainer *ECSContainer) (utils.Vec2f, error) {
	return ecsContainer.HitboxColliderManager.BaseColliderManager.GetCenter(e, ecsContainer)
}

func (hitboxColliderManager) GetLayer(e common.EntityId, ecsContainer *ECSContainer) (LayerMask, error) {
	return ecsContainer.HitboxColliderManager.BaseColliderManager.GetLayer(e, ecsContainer)
}

func (hitboxColliderManager) GetMask(e common.EntityId, ecsContainer *ECSContainer) (LayerMask, error) {
	return ecsContainer.HitboxColliderManager.BaseColliderManager.GetMask(e, ecsContainer)
}
