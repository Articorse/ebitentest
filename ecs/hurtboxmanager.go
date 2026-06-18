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

func (hurtboxColliderManager) EntityIds(world *World) []common.EntityId {
	return world.HurtboxColliders.GetEntities()
}

func (hurtboxColliderManager) HasCollider(e common.EntityId, world *World) bool {
	return world.HurtboxColliders.HasComponent(e)
}

func (hurtboxColliderManager) IsEnabled(e common.EntityId, world *World) (bool, error) {
	return world.HurtboxColliderManager.BaseColliderManager.IsEnabled(e, world)
}

func (hurtboxColliderManager) SetEnabled(e common.EntityId, enabled bool, world *World) error {
	return world.HurtboxColliderManager.BaseColliderManager.SetEnabled(e, enabled, world)
}

func (hurtboxColliderManager) GetWorldPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return world.HurtboxColliderManager.BaseColliderManager.GetWorldPaddedAABB(e, world)
}

func (hurtboxColliderManager) GetShapes(e common.EntityId, world *World) ([]shapes.Shape, error) {
	return world.HurtboxColliderManager.BaseColliderManager.GetShapes(e, world)
}

func (hurtboxColliderManager) GetLocalAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return world.HurtboxColliderManager.BaseColliderManager.GetLocalAABB(e, world)
}

func (hurtboxColliderManager) GetLocalPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return world.HurtboxColliderManager.BaseColliderManager.GetLocalPaddedAABB(e, world)
}

func (hurtboxColliderManager) GetCenter(e common.EntityId, world *World) (utils.Vec2, error) {
	return world.HurtboxColliderManager.BaseColliderManager.GetCenter(e, world)
}

func (hurtboxColliderManager) GetLayer(e common.EntityId, world *World) (LayerMask, error) {
	return world.HurtboxColliderManager.BaseColliderManager.GetLayer(e, world)
}

func (hurtboxColliderManager) GetMask(e common.EntityId, world *World) (LayerMask, error) {
	return world.HurtboxColliderManager.BaseColliderManager.GetMask(e, world)
}
