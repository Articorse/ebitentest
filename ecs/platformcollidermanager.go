package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
)

type platformColliderManager struct {
	BaseColliderManager[*platformCollider]
}

func NewPlatformColliderComponent(
	collisionLayer LayerMask,
	collisionMask LayerMask,
	shapes []shapes.Shape,
) *platformCollider {
	return &platformCollider{baseCollider: newBaseCollider(shapes, collisionLayer, collisionMask)}
}

func (platformColliderManager) EntityIds(ecsContainer *ECSContainer) []common.EntityId {
	return ecsContainer.PlatformColliders.GetEntities()
}

func (platformColliderManager) HasCollider(e common.EntityId, ecsContainer *ECSContainer) bool {
	return ecsContainer.PlatformColliders.HasComponent(e)
}

func (platformColliderManager) IsEnabled(e common.EntityId, ecsContainer *ECSContainer) (bool, error) {
	return platformColliderManager{}.BaseColliderManager.IsEnabled(e, ecsContainer)
}

func (platformColliderManager) SetEnabled(e common.EntityId, enabled bool, ecsContainer *ECSContainer) error {
	return platformColliderManager{}.BaseColliderManager.SetEnabled(e, enabled, ecsContainer)
}

func (platformColliderManager) GetecsContainerPaddedAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, ecsContainer)
}

func (platformColliderManager) GetShapes(e common.EntityId, ecsContainer *ECSContainer) ([]shapes.Shape, error) {
	return platformColliderManager{}.BaseColliderManager.GetShapes(e, ecsContainer)
}

func (platformColliderManager) GetLocalAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetLocalAABB(e, ecsContainer)
}

func (platformColliderManager) GetLocalPaddedAABB(e common.EntityId, ecsContainer *ECSContainer) ([2]utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, ecsContainer)
}

func (platformColliderManager) GetCenter(e common.EntityId, ecsContainer *ECSContainer) (utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetCenter(e, ecsContainer)
}

func (platformColliderManager) GetLayer(e common.EntityId, ecsContainer *ECSContainer) (LayerMask, error) {
	return platformColliderManager{}.BaseColliderManager.GetLayer(e, ecsContainer)
}

func (platformColliderManager) GetMask(e common.EntityId, ecsContainer *ECSContainer) (LayerMask, error) {
	return platformColliderManager{}.BaseColliderManager.GetMask(e, ecsContainer)
}
