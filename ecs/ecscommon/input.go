package ecscommon

import "github.com/hajimehoshi/ebiten/v2"

type InputState struct {
	Up, Down, Left, Right bool
}

type InputConfig struct {
	Up, Down, Left, Right ebiten.Key
	InputSourceFunc InputSourceFunc
}

type InputSourceFunc func(playerId PlayerId, tick uint64) InputState
