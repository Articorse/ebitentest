package common

import "ebittest/utils"

type TickState struct {
	Collisions        []Collision
	AABBCollisions    []EntityPair
	ProximateEntities []EntityPair
	CollisionGrid     map[utils.Vec2i][]EntityId
}

func NewTickState() *TickState {
	return &TickState{
		CollisionGrid: make(map[utils.Vec2i][]EntityId),
	}
}
