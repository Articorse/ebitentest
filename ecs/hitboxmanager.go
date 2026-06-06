package ecs

import (
	"ebittest/ecs/collidershapes"
	"ebittest/ecs/common"
	"ebittest/utils"
)

type HitboxColliderManager struct {
	BaseColliderManager[*hitboxCollider]
}

func NewHitboxColliderComponent(
	shapes []collidershapes.Shape,
) *hitboxCollider {
	return &hitboxCollider{baseCollider: newBaseCollider(shapes)}
}

func (HitboxColliderManager) EntityIds(w *World) []common.EntityId {
	ids := make([]common.EntityId, 0, len(w.HitboxColliders))
	for e := range w.HitboxColliders {
		ids = append(ids, e)
	}
	return ids
}

func (HitboxColliderManager) HasCollider(e common.EntityId, w *World) bool {
	_, ok := w.HitboxColliders[e]
	return ok
}

func (HitboxColliderManager) GetWorldPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, w.HitboxColliders, w.Transforms, w.Parents)
}

func (HitboxColliderManager) GetShapes(e common.EntityId, w *World) ([]collidershapes.Shape, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetShapes(e, w.HitboxColliders)
}

func (HitboxColliderManager) GetLocalAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetLocalAABB(e, w.HitboxColliders)
}

func (HitboxColliderManager) GetLocalPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, w.HitboxColliders)
}

func (HitboxColliderManager) GetCenter(e common.EntityId, w *World) (utils.Vec2, error) {
	return HitboxColliderManager{}.BaseColliderManager.GetCenter(e, w.HitboxColliders)
}
