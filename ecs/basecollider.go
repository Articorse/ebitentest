package ecs

import (
	"ebittest/ecs/collidershapes"
	"ebittest/utils"
)

type baseCollider struct {
	shapes     []collidershapes.Shape
	center     utils.Vec2
	aabb       [2]utils.Vec2
	paddedAabb [2]utils.Vec2
}
