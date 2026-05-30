package components

import (
	"ebittest/ecs/components/hitboxes"
	"ebittest/utils"
)

type ColliderType uint8
type ColliderLayer uint16

const (
	Collider_Mob ColliderType = iota
	Collider_Static
	Collider_Trigger

	Layer_Player             = ColliderLayer(0b1000000000000000)
	Layer_Enemy              = ColliderLayer(0b0100000000000000)
	Layer_FriendlyProjectile = ColliderLayer(0b0010000000000000)
	Layer_EnemyProjectile    = ColliderLayer(0b0001000000000000)
	Layer_Terrain            = ColliderLayer(0b0000100000000000)
	Layer_Platform           = ColliderLayer(0b0000010000000000)
)

type Collider struct {
	colliderType ColliderType
	hitboxes     []hitboxes.Hitbox
	layers       ColliderLayer
	mask         ColliderLayer
	center       utils.Vec2
	aabb         [2]utils.Vec2
	paddedAabb   [2]utils.Vec2
}

func (Collider) isComponent() {}
