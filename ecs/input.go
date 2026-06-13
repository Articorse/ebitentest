package ecs

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type RotationInput uint8

type InputConfig struct {
	Up, Down, Left, Right, Dodge ebiten.Key
	Use                          ebiten.MouseButton
}

type input struct {
	config          InputConfig
	inputSourceFunc InputSourceFunc
}

func (input) isComponent() {}

func (x input) Copy() input {
	return input{
		config:          x.config,
		inputSourceFunc: x.inputSourceFunc,
	}
}
