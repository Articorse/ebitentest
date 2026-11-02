package common

import "ebittest/utils"

type TickState struct {
	CollisionGrid map[utils.Vec2i][]EntityId
}

func NewTickState() *TickState {
	return &TickState{
		CollisionGrid: make(map[utils.Vec2i][]EntityId),
	}
}
