package ecscommon

import "ebittest/utils"

type InputState struct {
	Up, Down, Left, Right bool
	MousePos              utils.Vec2
	Use                   bool
}

type InputSourceFunc func(entityId EntityId, tick uint64) InputState
