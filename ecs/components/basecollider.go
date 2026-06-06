package components

import (
	"ebittest/ecs/components/collidershapes"
	"ebittest/utils"
)

type BaseCollider struct {
	shapes     []collidershapes.Shape
	center     utils.Vec2
	aabb       [2]utils.Vec2
	paddedAabb [2]utils.Vec2
}
