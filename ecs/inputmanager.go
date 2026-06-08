package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type InputManager struct{}

type InputState struct {
	Analog1X, Analog1Y float64
	MouseScreenPos     utils.Vec2
	Use                bool
}

type InputSourceFunc func(
	e common.EntityId,
	tick uint64,
	world *World,
) InputState

func NewInputComponent(config InputConfig, inputSourceFunc InputSourceFunc) *input {
	return &input{config: config, inputSourceFunc: inputSourceFunc}
}

func (*InputManager) GetInputConfig(
	e common.EntityId,
	inputs map[common.EntityId]*input,
) (InputConfig, error) {
	isf, ok := inputs[e]
	if !ok {
		return InputConfig{}, fmt.Errorf("could not get input of entity %d", e)
	}

	return isf.config, nil
}

func (*InputManager) SetInputConfig(
	e common.EntityId,
	config InputConfig,
	inputs map[common.EntityId]*input,
) error {
	isf, ok := inputs[e]
	if !ok {
		return fmt.Errorf("could not get input of entity %d", e)
	}

	isf.config = config

	return nil
}

func (*InputManager) GetInputSourceFunc(
	e common.EntityId,
	inputs map[common.EntityId]*input,
) (InputSourceFunc, error) {
	isf, ok := inputs[e]
	if !ok {
		return nil, fmt.Errorf("could not get input of entity %d", e)
	}

	return isf.inputSourceFunc, nil
}

func (*InputManager) SetInputSourceFunc(
	e common.EntityId,
	inputSourceFunc InputSourceFunc,
	inputs map[common.EntityId]*input,
) error {
	isf, ok := inputs[e]
	if !ok {
		return fmt.Errorf("could not get input of entity %d", e)
	}

	isf.inputSourceFunc = inputSourceFunc

	return nil
}
