package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
)

type HitboxColliderManager struct {
	BaseColliderManager[*hitboxCollider]
}

func NewHitboxColliderComponent(
	collisionLayer LayerMask,
	collisionMask LayerMask,
	shapes ...shapes.Shape,
) *hitboxCollider {
	return &hitboxCollider{baseCollider: newBaseCollider(shapes, collisionLayer, collisionMask)}
}

func (HitboxColliderManager) EntityIds(world *World) []common.EntityId {
	return world.HitboxColliders.GetEntities()
}

func (HitboxColliderManager) HasCollider(e common.EntityId, world *World) bool {
	return world.HitboxColliders.HasComponent(e)
}

func (HitboxColliderManager) GetWorldPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, world)
}

func (HitboxColliderManager) GetShapes(e common.EntityId, world *World) ([]shapes.Shape, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetShapes(e, world)
}

func (HitboxColliderManager) GetLocalAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetLocalAABB(e, world)
}

func (HitboxColliderManager) GetLocalPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, world)
}

func (HitboxColliderManager) GetCenter(e common.EntityId, world *World) (utils.Vec2, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetCenter(e, world)
}

func (HitboxColliderManager) GetLayer(e common.EntityId, world *World) (LayerMask, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetLayer(e, world)
}

func (HitboxColliderManager) GetMask(e common.EntityId, world *World) (LayerMask, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetMask(e, world)
}
