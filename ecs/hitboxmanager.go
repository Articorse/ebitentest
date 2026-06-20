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

func (hitboxColliderManager) EntityIds(ecs *ECS) []common.EntityId {
	return ecs.HitboxColliders.GetEntities()
}

func (hitboxColliderManager) HasCollider(e common.EntityId, ecs *ECS) bool {
	return ecs.HitboxColliders.HasComponent(e)
}

func (hitboxColliderManager) IsEnabled(e common.EntityId, ecs *ECS) (bool, error) {
	return ecs.HitboxColliderManager.BaseColliderManager.IsEnabled(e, ecs)
}

func (hitboxColliderManager) SetEnabled(e common.EntityId, enabled bool, ecs *ECS) error {
	return ecs.HitboxColliderManager.BaseColliderManager.SetEnabled(e, enabled, ecs)
}

func (hitboxColliderManager) GetWorldPaddedAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return ecs.HitboxColliderManager.BaseColliderManager.GetWorldPaddedAABB(e, ecs)
}

func (hitboxColliderManager) GetShapes(e common.EntityId, ecs *ECS) ([]shapes.Shape, error) {
	return ecs.HitboxColliderManager.BaseColliderManager.GetShapes(e, ecs)
}

func (hitboxColliderManager) GetLocalAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return ecs.HitboxColliderManager.BaseColliderManager.GetLocalAABB(e, ecs)
}

func (hitboxColliderManager) GetLocalPaddedAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return ecs.HitboxColliderManager.BaseColliderManager.GetLocalPaddedAABB(e, ecs)
}

func (hitboxColliderManager) GetCenter(e common.EntityId, ecs *ECS) (utils.Vec2, error) {
	return ecs.HitboxColliderManager.BaseColliderManager.GetCenter(e, ecs)
}

func (hitboxColliderManager) GetLayer(e common.EntityId, ecs *ECS) (LayerMask, error) {
	return ecs.HitboxColliderManager.BaseColliderManager.GetLayer(e, ecs)
}

func (hitboxColliderManager) GetMask(e common.EntityId, ecs *ECS) (LayerMask, error) {
	return ecs.HitboxColliderManager.BaseColliderManager.GetMask(e, ecs)
}
