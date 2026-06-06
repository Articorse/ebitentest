package components

import (
	"ebittest/ecs/components/collidershapes"
)

type HitboxColliderManager struct {
	BaseColliderManager[*HitboxCollider]
}

func NewHitboxColliderComponent(
	shapes []collidershapes.Shape,
) *HitboxCollider {
	return &HitboxCollider{BaseCollider: newBaseCollider(shapes)}
}
