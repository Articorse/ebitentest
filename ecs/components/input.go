package components

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type RotationInput uint8

type InputConfig struct {
	Up, Down, Left, Right ebiten.Key
	Use                   ebiten.MouseButton
}

type Input struct {
	config          InputConfig
	inputSourceFunc InputSourceFunc
}

func (Input) isComponent() {}

func (x Input) Copy() Input {
	return Input{
		config:          x.config,
		inputSourceFunc: x.inputSourceFunc,
	}
}
