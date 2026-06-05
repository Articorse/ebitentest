package components

import (
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
)

type InputManager struct{}

type InputState struct {
	Up, Down, Left, Right bool
	MouseScreenPos        utils.Vec2
	Use                   bool
}

type InputSourceFunc func(
	e ecscommon.EntityId,
	tick uint64,
	inputs map[ecscommon.EntityId]*Input,
) InputState

func NewInputComponent(config InputConfig, inputSourceFunc InputSourceFunc) *Input {
	return &Input{config: config, inputSourceFunc: inputSourceFunc}
}

func (*InputManager) GetInputConfig(
	e ecscommon.EntityId,
	inputs map[ecscommon.EntityId]*Input,
) (InputConfig, error) {
	isf, ok := inputs[e]
	if !ok {
		return InputConfig{}, fmt.Errorf("could not get input of entity %d", e)
	}

	return isf.config, nil
}

func (*InputManager) SetInputConfig(
	e ecscommon.EntityId,
	config InputConfig,
	inputs map[ecscommon.EntityId]*Input,
) error {
	isf, ok := inputs[e]
	if !ok {
		return fmt.Errorf("could not get input of entity %d", e)
	}

	isf.config = config

	return nil
}

func (*InputManager) GetInputSourceFunc(
	e ecscommon.EntityId,
	inputs map[ecscommon.EntityId]*Input,
) (InputSourceFunc, error) {
	isf, ok := inputs[e]
	if !ok {
		return nil, fmt.Errorf("could not get input of entity %d", e)
	}

	return isf.inputSourceFunc, nil
}

func (*InputManager) SetInputSourceFunc(
	e ecscommon.EntityId,
	inputSourceFunc InputSourceFunc,
	inputs map[ecscommon.EntityId]*Input,
) error {
	isf, ok := inputs[e]
	if !ok {
		return fmt.Errorf("could not get input of entity %d", e)
	}

	isf.inputSourceFunc = inputSourceFunc

	return nil
}
