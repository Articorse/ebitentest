package ecs

import (
	"ebittest/ecs/shapes"
	"ebittest/utils"
)

type baseCollider struct {
	shapes     []shapes.Shape
	center     utils.Vec2
	aabb       [2]utils.Vec2
	paddedAabb [2]utils.Vec2
}
