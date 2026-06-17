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

func (hitboxColliderManager) GetWorldPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return hitboxColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, world)
}

func (hitboxColliderManager) GetShapes(e common.EntityId, world *World) ([]shapes.Shape, error) {
	return hitboxColliderManager{}.BaseColliderManager.GetShapes(e, world)
}

func (hitboxColliderManager) GetLocalAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return hitboxColliderManager{}.BaseColliderManager.GetLocalAABB(e, world)
}

func (hitboxColliderManager) GetLocalPaddedAABB(e common.EntityId, world *World) ([2]utils.Vec2, error) {
	return hitboxColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, world)
}

func (hitboxColliderManager) GetCenter(e common.EntityId, world *World) (utils.Vec2, error) {
	return hitboxColliderManager{}.BaseColliderManager.GetCenter(e, world)
}

func (hitboxColliderManager) GetLayer(e common.EntityId, world *World) (LayerMask, error) {
	return hitboxColliderManager{}.BaseColliderManager.GetLayer(e, world)
}

func (hitboxColliderManager) GetMask(e common.EntityId, world *World) (LayerMask, error) {
	return hitboxColliderManager{}.BaseColliderManager.GetMask(e, world)
}
