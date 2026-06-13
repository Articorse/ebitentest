package ecs

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type InputType uint64

const (
	Input_Analog1Y InputType = iota
	Input_Analog1X
	Input_Analog2Y
	Input_Analog2X
	Input_Dodge
	Input_Ability1
	Input_Ability2
)

type InputKey struct {
	gamepadId   *ebiten.GamepadID
	keyboardKey [2]*ebiten.Key
	mouseKey    [2]*ebiten.MouseButton
	gamepadKey  [2]*ebiten.GamepadButton
	gamepadAxis *ebiten.GamepadAxisType
}

func NewKeyboardInputKey(valueUp, valueDown *ebiten.Key) InputKey {
	return InputKey{
		keyboardKey: [2]*ebiten.Key{valueUp, valueDown},
	}
}

func NewMouseInputKey(valueUp, valueDown *ebiten.MouseButton) InputKey {
	return InputKey{
		mouseKey: [2]*ebiten.MouseButton{valueUp, valueDown},
	}
}

func NewGamepadButtonInputKey(gamepadId ebiten.GamepadID, valueUp, valueDown *ebiten.GamepadButton) InputKey {
	return InputKey{
		gamepadId:  &gamepadId,
		gamepadKey: [2]*ebiten.GamepadButton{valueUp, valueDown},
	}
}

func NewGamepadAxisInputKey(gamepadId ebiten.GamepadID, axis ebiten.GamepadAxisType) InputKey {
	return InputKey{
		gamepadId:   &gamepadId,
		gamepadAxis: &axis,
	}
}

func (x InputKey) GetInput() float64 {
	if x.keyboardKey[0] != nil {
		if ebiten.IsKeyPressed(*x.keyboardKey[0]) {
			return 1
		}
	}
	if x.keyboardKey[1] != nil {
		if ebiten.IsKeyPressed(*x.keyboardKey[1]) {
			return -1
		}
	}
	if x.mouseKey[0] != nil {
		if ebiten.IsMouseButtonPressed(*x.mouseKey[0]) {
			return 1
		}
	}
	if x.mouseKey[1] != nil {
		if ebiten.IsMouseButtonPressed(*x.mouseKey[1]) {
			return -1
		}
	}
	if x.gamepadKey[0] != nil {
		if ebiten.IsGamepadButtonPressed(*x.gamepadId, *x.gamepadKey[0]) {
			return 1
		}
	}
	if x.gamepadKey[1] != nil {
		if ebiten.IsGamepadButtonPressed(*x.gamepadId, *x.gamepadKey[1]) {
			return -1
		}
	}
	if x.gamepadAxis != nil {
		return ebiten.GamepadAxisValue(*x.gamepadId, *x.gamepadAxis)
	}

	return 0
}

type input struct {
	config          map[InputType]InputKey
	inputSourceFunc InputSourceFunc
}

func (input) isComponent() {}

func (x input) Copy() input {
	return input{
		config:          x.config,
		inputSourceFunc: x.inputSourceFunc,
	}
}
