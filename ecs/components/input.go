package components

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type InputConfig struct {
	Up, Down, Left, Right ebiten.Key
	Use                   ebiten.MouseButton
}

type Input struct {
	config          InputConfig
	inputSourceFunc InputSourceFunc
}

func (Input) isComponent() {}
