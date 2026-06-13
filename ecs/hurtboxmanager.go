package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
)

type HurtboxColliderManager struct {
	BaseColliderManager[*hurtboxCollider]
}

func NewHurtboxColliderComponent(
	collisionLayer LayerMask,
	collisionMask LayerMask,
	shapes ...shapes.Shape,
) *hurtboxCollider {
	return &hurtboxCollider{baseCollider: newBaseCollider(shapes, collisionLayer, collisionMask)}
}

func (HurtboxColliderManager) EntityIds(world *World) []common.EntityId {
	return world.HurtboxColliders.GetEntities()
}

func (HurtboxColliderManager) HasCollider(e common.EntityId, world *World) bool {
	return world.HurtboxColliders.HasComponent(e)
}

func (HurtboxColliderManager) GetWorldPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, world)
}

func (HurtboxColliderManager) GetShapes(e common.EntityId, world *World) ([]shapes.Shape, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetShapes(e, world)
}

func (HurtboxColliderManager) GetLocalAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetLocalAABB(e, world)
}

func (HurtboxColliderManager) GetLocalPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, world)
}

func (HurtboxColliderManager) GetCenter(e common.EntityId, world *World) (utils.Vec2, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetCenter(e, world)
}

func (HurtboxColliderManager) GetLayer(e common.EntityId, world *World) (LayerMask, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetLayer(e, world)
}

func (HurtboxColliderManager) GetMask(e common.EntityId, world *World) (LayerMask, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetMask(e, world)
}
