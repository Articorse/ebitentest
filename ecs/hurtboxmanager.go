package ecs

import (
	"ebittest/ecs/shapes"
	"ebittest/ecs/common"
	"ebittest/utils"
)

type HurtboxColliderManager struct {
	BaseColliderManager[*hurtboxCollider]
}

func NewHurtboxColliderComponent(
	shapes ...shapes.Shape,
) *hurtboxCollider {
	return &hurtboxCollider{baseCollider: newBaseCollider(shapes)}
}

func (HurtboxColliderManager) EntityIds(w *World) []common.EntityId {
	ids := make([]common.EntityId, 0, len(w.HurtboxColliders))
	for e := range w.HurtboxColliders {
		ids = append(ids, e)
	}
	return ids
}

func (HurtboxColliderManager) HasCollider(e common.EntityId, w *World) bool {
	_, ok := w.HurtboxColliders[e]
	return ok
}

func (HurtboxColliderManager) GetWorldPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, w.HurtboxColliders, w.Transforms, w.Parents)
}

func (HurtboxColliderManager) GetShapes(e common.EntityId, w *World) ([]shapes.Shape, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetShapes(e, w.HurtboxColliders)
}

func (HurtboxColliderManager) GetLocalAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetLocalAABB(e, w.HurtboxColliders)
}

func (HurtboxColliderManager) GetLocalPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, w.HurtboxColliders)
}

func (HurtboxColliderManager) GetCenter(e common.EntityId, w *World) (utils.Vec2, error) {
	return HurtboxColliderManager{}.BaseColliderManager.GetCenter(e, w.HurtboxColliders)
}
