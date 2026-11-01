package common

import "ebittest/utils"

type EntityPair struct {
	A EntityId
	B EntityId
}

type Collision struct {
	A         EntityId
	B         EntityId
	Vector    utils.Vec2f
	AShapeIdx int
	BShapeIdx int
}
