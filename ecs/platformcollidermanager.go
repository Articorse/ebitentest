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

func (platformColliderManager) EntityIds(world *World) []common.EntityId {
	return world.PlatformColliders.GetEntities()
}

func (platformColliderManager) HasCollider(e common.EntityId, world *World) bool {
	return world.PlatformColliders.HasComponent(e)
}

func (platformColliderManager) IsEnabled(e common.EntityId, world *World) (bool, error) {
	return platformColliderManager{}.BaseColliderManager.IsEnabled(e, world)
}

func (platformColliderManager) SetEnabled(e common.EntityId, enabled bool, world *World) error {
	return platformColliderManager{}.BaseColliderManager.SetEnabled(e, enabled, world)
}

func (platformColliderManager) GetWorldPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, world)
}

func (platformColliderManager) GetShapes(e common.EntityId, world *World) ([]shapes.Shape, error) {
	return platformColliderManager{}.BaseColliderManager.GetShapes(e, world)
}

func (platformColliderManager) GetLocalAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetLocalAABB(e, world)
}

func (platformColliderManager) GetLocalPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, world)
}

func (platformColliderManager) GetCenter(e common.EntityId, world *World) (utils.Vec2, error) {
	return platformColliderManager{}.BaseColliderManager.GetCenter(e, world)
}

func (platformColliderManager) GetLayer(e common.EntityId, world *World) (LayerMask, error) {
	return platformColliderManager{}.BaseColliderManager.GetLayer(e, world)
}

func (platformColliderManager) GetMask(e common.EntityId, world *World) (LayerMask, error) {
	return platformColliderManager{}.BaseColliderManager.GetMask(e, world)
}
