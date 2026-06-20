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

func (hurtboxColliderManager) EntityIds(ecs *ECS) []common.EntityId {
	return ecs.HurtboxColliders.GetEntities()
}

func (hurtboxColliderManager) HasCollider(e common.EntityId, ecs *ECS) bool {
	return ecs.HurtboxColliders.HasComponent(e)
}

func (hurtboxColliderManager) IsEnabled(e common.EntityId, ecs *ECS) (bool, error) {
	return ecs.HurtboxColliderManager.BaseColliderManager.IsEnabled(e, ecs)
}

func (hurtboxColliderManager) SetEnabled(e common.EntityId, enabled bool, ecs *ECS) error {
	return ecs.HurtboxColliderManager.BaseColliderManager.SetEnabled(e, enabled, ecs)
}

func (hurtboxColliderManager) GetWorldPaddedAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return ecs.HurtboxColliderManager.BaseColliderManager.GetWorldPaddedAABB(e, ecs)
}

func (hurtboxColliderManager) GetShapes(e common.EntityId, ecs *ECS) ([]shapes.Shape, error) {
	return ecs.HurtboxColliderManager.BaseColliderManager.GetShapes(e, ecs)
}

func (hurtboxColliderManager) GetLocalAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return ecs.HurtboxColliderManager.BaseColliderManager.GetLocalAABB(e, ecs)
}

func (hurtboxColliderManager) GetLocalPaddedAABB(e common.EntityId, ecs *ECS) ([2]utils.Vec2, error) {
	return ecs.HurtboxColliderManager.BaseColliderManager.GetLocalPaddedAABB(e, ecs)
}

func (hurtboxColliderManager) GetCenter(e common.EntityId, ecs *ECS) (utils.Vec2, error) {
	return ecs.HurtboxColliderManager.BaseColliderManager.GetCenter(e, ecs)
}

func (hurtboxColliderManager) GetLayer(e common.EntityId, ecs *ECS) (LayerMask, error) {
	return ecs.HurtboxColliderManager.BaseColliderManager.GetLayer(e, ecs)
}

func (hurtboxColliderManager) GetMask(e common.EntityId, ecs *ECS) (LayerMask, error) {
	return ecs.HurtboxColliderManager.BaseColliderManager.GetMask(e, ecs)
}
