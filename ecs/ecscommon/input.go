package ecscommon

type InputState struct {
	Up, Down, Left, Right bool
}

type InputSourceFunc func(playerId PlayerId, tick uint64) InputState
