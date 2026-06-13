package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
)

type PlatformColliderManager struct {
	BaseColliderManager[*platformCollider]
}

func NewPlatformColliderComponent(
	collisionLayer LayerMask,
	collisionMask LayerMask,
	shapes []shapes.Shape,
) *platformCollider {
	return &platformCollider{baseCollider: newBaseCollider(shapes, collisionLayer, collisionMask)}
}

func (PlatformColliderManager) EntityIds(world *World) []common.EntityId {
	return world.PlatformColliders.GetEntities()
}

func (PlatformColliderManager) HasCollider(e common.EntityId, world *World) bool {
	return world.PlatformColliders.HasComponent(e)
}

func (PlatformColliderManager) GetWorldPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return PlatformColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, world)
}

func (PlatformColliderManager) GetShapes(e common.EntityId, world *World) ([]shapes.Shape, error) {
	return PlatformColliderManager{}.BaseColliderManager.GetShapes(e, world)
}

func (PlatformColliderManager) GetLocalAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return PlatformColliderManager{}.BaseColliderManager.GetLocalAABB(e, world)
}

func (PlatformColliderManager) GetLocalPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return PlatformColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, world)
}

func (PlatformColliderManager) GetCenter(e common.EntityId, world *World) (utils.Vec2, error) {
	return PlatformColliderManager{}.BaseColliderManager.GetCenter(e, world)
}

func (PlatformColliderManager) GetLayer(e common.EntityId, world *World) (LayerMask, error) {
	return PlatformColliderManager{}.BaseColliderManager.GetLayer(e, world)
}

func (PlatformColliderManager) GetMask(e common.EntityId, world *World) (LayerMask, error) {
	return PlatformColliderManager{}.BaseColliderManager.GetMask(e, world)
}
