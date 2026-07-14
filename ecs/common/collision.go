package common

import "ebittest/utils"

type Collision struct {
	Vector    utils.Vec2f
	AShapeIdx int
	BShapeIdx int
}
