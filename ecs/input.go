package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"maps"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The IDs are defined by the content that registers them
type InputTypeEnum uint8

type InputType uint64

const (
	Input_Analog1Y InputType = iota
	Input_Analog1X
	Input_Analog2Y
	Input_Analog2X
	Input_MainHandAbility1
	Input_MainHandAbility2
	Input_OffHandAbility1
	Input_OffHandAbility2
	Input_Ability1
	Input_Ability2
)

type InputSourceFunc func(
	e common.EntityId,
	tick uint64,
	inputParams InputParams,
	ecsContainer *ECSContainer,
) (InputState, error)

type InputParams interface {
	IsInputParams()
	Copy() InputParams
}

type InputKey struct {
	GamepadId   ebiten.GamepadID
	KeyboardKey [2]ebiten.Key
	MouseKey    [2]ebiten.MouseButton
	GamepadKey  [2]ebiten.StandardGamepadButton
	GamepadAxis ebiten.StandardGamepadAxis
}

func NewKeyboardInputKey(valueUp, valueDown ebiten.Key) InputKey {
	k := newUnsetInputKey()
	k.KeyboardKey = [2]ebiten.Key{valueUp, valueDown}
	return k
}

func NewMouseInputKey(valueUp, valueDown ebiten.MouseButton) InputKey {
	k := newUnsetInputKey()
	k.MouseKey = [2]ebiten.MouseButton{valueUp, valueDown}
	return k
}

func NewGamepadButtonInputKey(gamepadId ebiten.GamepadID, valueUp, valueDown ebiten.StandardGamepadButton) InputKey {
	k := newUnsetInputKey()
	k.GamepadId = gamepadId
	k.GamepadKey = [2]ebiten.StandardGamepadButton{valueUp, valueDown}
	return k
}

func NewGamepadAxisInputKey(gamepadId ebiten.GamepadID, axis ebiten.StandardGamepadAxis) InputKey {
	k := newUnsetInputKey()
	k.GamepadId = gamepadId
	k.GamepadAxis = axis
	return k
}

func (x InputKey) GetInput() float64 {
	if x.KeyboardKey[0] != -1 {
		if ebiten.IsKeyPressed(x.KeyboardKey[0]) {
			return 1
		}
	}
	if x.KeyboardKey[1] != -1 {
		if ebiten.IsKeyPressed(x.KeyboardKey[1]) {
			return -1
		}
	}
	if x.MouseKey[0] != -1 {
		if ebiten.IsMouseButtonPressed(x.MouseKey[0]) {
			return 1
		}
	}
	if x.MouseKey[1] != -1 {
		if ebiten.IsMouseButtonPressed(x.MouseKey[1]) {
			return -1
		}
	}
	if x.GamepadKey[0] != -1 {
		if ebiten.IsStandardGamepadButtonPressed(x.GamepadId, x.GamepadKey[0]) {
			return 1
		}
	}
	if x.GamepadKey[1] != -1 {
		if ebiten.IsStandardGamepadButtonPressed(x.GamepadId, x.GamepadKey[1]) {
			return -1
		}
	}
	if x.GamepadAxis != -1 {
		axisVal := ebiten.StandardGamepadAxisValue(x.GamepadId, x.GamepadAxis)
		if math.Abs(axisVal) > data.GamepadDeadzone {
			return axisVal
		} else {
			return 0
		}
	}

	return 0
}

type input struct {
	config        map[InputType]InputKey
	inputType     InputTypeEnum
	facingInput   FacingInputEnum
	params        InputParams
	lastFacingDir utils.Vec2f
}

func newUnsetInputKey() InputKey {
	return InputKey{
		KeyboardKey: [2]ebiten.Key{-1, -1},
		MouseKey:    [2]ebiten.MouseButton{-1, -1},
		GamepadKey:  [2]ebiten.StandardGamepadButton{-1, -1},
		GamepadAxis: -1,
	}
}

func (x input) Copy() input {
	return input{
		config:        x.config,
		inputType:     x.inputType,
		facingInput:   x.facingInput,
		params:        copyInputParams(x.params),
		lastFacingDir: utils.Vec2f{},
	}
}

func (x input) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Inputs.addComponent(e, x.Copy())
}

type inputDto struct {
	Config        map[InputType]InputKey
	InputType     InputTypeEnum
	FacingInput   FacingInputEnum
	Params        InputParams
	LastFacingDir utils.Vec2f
}

func (x input) toDto() (ComponentDto, error) {
	return &inputDto{
		Config:        maps.Clone(x.config),
		InputType:     x.inputType,
		FacingInput:   x.facingInput,
		Params:        copyInputParams(x.params),
		LastFacingDir: x.lastFacingDir,
	}, nil
}

func copyInputParams(params InputParams) InputParams {
	if params == nil {
		return nil
	}
	return params.Copy()
}

func (x *inputDto) toComponent() (Component, error) {
	return &input{
		config:        x.Config,
		inputType:     x.InputType,
		facingInput:   x.FacingInput,
		params:        x.Params,
		lastFacingDir: x.LastFacingDir,
	}, nil
}
