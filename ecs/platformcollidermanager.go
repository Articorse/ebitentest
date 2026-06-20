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

func (platformColliderManager) EntityIds(ecs *ECS) []common.EntityId {
	return ecs.PlatformColliders.GetEntities()
}

func (platformColliderManager) HasCollider(e common.EntityId, ecs *ECS) bool {
	return ecs.PlatformColliders.HasComponent(e)
}

func (platformColliderManager) IsEnabled(e common.EntityId, ecs *ECS) (bool, error) {
	return platformColliderManager{}.BaseColliderManager.IsEnabled(e, ecs)
}

func (platformColliderManager) SetEnabled(e common.EntityId, enabled bool, ecs *ECS) error {
	return platformColliderManager{}.BaseColliderManager.SetEnabled(e, enabled, ecs)
}

func (platformColliderManager) GetWorldPaddedAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, ecs)
}

func (platformColliderManager) GetShapes(e common.EntityId, ecs *ECS) ([]shapes.Shape, error) {
	return platformColliderManager{}.BaseColliderManager.GetShapes(e, ecs)
}

func (platformColliderManager) GetLocalAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetLocalAABB(e, ecs)
}

func (platformColliderManager) GetLocalPaddedAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, ecs)
}

func (platformColliderManager) GetCenter(e common.EntityId, ecs *ECS) (utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetCenter(e, ecs)
}

func (platformColliderManager) GetLayer(e common.EntityId, ecs *ECS) (LayerMask, error) {
	return platformColliderManager{}.BaseColliderManager.GetLayer(e, ecs)
}

func (platformColliderManager) GetMask(e common.EntityId, ecs *ECS) (LayerMask, error) {
	return platformColliderManager{}.BaseColliderManager.GetMask(e, ecs)
}
