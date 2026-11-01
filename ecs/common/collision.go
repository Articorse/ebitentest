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

// One of an entity's collider shapes overlapping one tile
type TileCollision struct {
	E        EntityId
	Tile     utils.Vec2i
	Vector   utils.Vec2f
	ShapeIdx int
}
