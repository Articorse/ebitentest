package components

import (
	"ebittest/ecs/components/hitboxes"
	"ebittest/utils"
)

type ColliderType uint8

const (
	Collider_Mob ColliderType = iota
	Collider_Static
	Collider_Trigger
)

type Collider struct {
	colliderType ColliderType
	hitboxes     []hitboxes.Hitbox
	center       utils.Vec2
	aabb         [2]utils.Vec2
}

func (Collider) isComponent() {}
