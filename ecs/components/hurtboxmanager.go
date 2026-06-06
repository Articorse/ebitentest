package components

import (
	"ebittest/ecs/components/collidershapes"
)

type HurtboxColliderManager struct {
	BaseColliderManager[*HurtboxCollider]
}

func NewHurtboxColliderComponent(
	shapes []collidershapes.Shape,
) *HurtboxCollider {
	return &HurtboxCollider{BaseCollider: newBaseCollider(shapes)}
}
