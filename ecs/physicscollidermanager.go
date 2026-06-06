package ecs

import (
	"ebittest/ecs/collidershapes"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type PhysicsColliderManager struct {
	BaseColliderManager[*physicsCollider]
}

func NewPhysicsColliderComponent(
	cType PhysicsColliderType,
	shapes []collidershapes.Shape,
) *physicsCollider {
	return &physicsCollider{
		colliderType: cType,
		baseCollider: newBaseCollider(shapes),
	}
}

func (*PhysicsColliderManager) GetColliderType(
	e common.EntityId,
	colliders map[common.EntityId]*physicsCollider,
) (PhysicsColliderType, error) {
	collider, ok := colliders[e]
	if !ok {
		return 0, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.colliderType, nil
}

func (PhysicsColliderManager) EntityIds(w *World) []common.EntityId {
	ids := make([]common.EntityId, 0, len(w.PhysicsColliders))
	for e := range w.PhysicsColliders {
		ids = append(ids, e)
	}
	return ids
}

func (PhysicsColliderManager) HasCollider(e common.EntityId, w *World) bool {
	_, ok := w.PhysicsColliders[e]
	return ok
}

func (PhysicsColliderManager) GetWorldPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return PhysicsColliderManager{}.BaseColliderManager.GetWorldPaddedAABB(e, w.PhysicsColliders, w.Transforms, w.Parents)
}

func (PhysicsColliderManager) GetShapes(e common.EntityId, w *World) ([]collidershapes.Shape, error) {
	return PhysicsColliderManager{}.BaseColliderManager.GetShapes(e, w.PhysicsColliders)
}

func (PhysicsColliderManager) GetLocalAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return PhysicsColliderManager{}.BaseColliderManager.GetLocalAABB(e, w.PhysicsColliders)
}

func (PhysicsColliderManager) GetLocalPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error) {
	return PhysicsColliderManager{}.BaseColliderManager.GetLocalPaddedAABB(e, w.PhysicsColliders)
}

func (PhysicsColliderManager) GetCenter(e common.EntityId, w *World) (utils.Vec2, error) {
	return PhysicsColliderManager{}.BaseColliderManager.GetCenter(e, w.PhysicsColliders)
}
