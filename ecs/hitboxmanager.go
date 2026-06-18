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

func (hitboxColliderManager) EntityIds(world *World) []common.EntityId {
	return world.HitboxColliders.GetEntities()
}

func (hitboxColliderManager) HasCollider(e common.EntityId, world *World) bool {
	return world.HitboxColliders.HasComponent(e)
}

func (hitboxColliderManager) IsEnabled(e common.EntityId, world *World) (bool, error) {
	return world.HitboxColliderManager.BaseColliderManager.IsEnabled(e, world)
}

func (hitboxColliderManager) SetEnabled(e common.EntityId, enabled bool, world *World) error {
	return world.HitboxColliderManager.BaseColliderManager.SetEnabled(e, enabled, world)
}

func (hitboxColliderManager) GetWorldPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return world.HitboxColliderManager.BaseColliderManager.GetWorldPaddedAABB(e, world)
}

func (hitboxColliderManager) GetShapes(e common.EntityId, world *World) ([]shapes.Shape, error) {
	return world.HitboxColliderManager.BaseColliderManager.GetShapes(e, world)
}

func (hitboxColliderManager) GetLocalAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return world.HitboxColliderManager.BaseColliderManager.GetLocalAABB(e, world)
}

func (hitboxColliderManager) GetLocalPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return world.HitboxColliderManager.BaseColliderManager.GetLocalPaddedAABB(e, world)
}

func (hitboxColliderManager) GetCenter(e common.EntityId, world *World) (utils.Vec2, error) {
	return world.HitboxColliderManager.BaseColliderManager.GetCenter(e, world)
}

func (hitboxColliderManager) GetLayer(e common.EntityId, world *World) (LayerMask, error) {
	return world.HitboxColliderManager.BaseColliderManager.GetLayer(e, world)
}

func (hitboxColliderManager) GetMask(e common.EntityId, world *World) (LayerMask, error) {
	return world.HitboxColliderManager.BaseColliderManager.GetMask(e, world)
}
