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
	world *World,
) (InputConfig, error) {
	isf, err := world.Inputs.getComponent(e)
	if err != nil {
		return InputConfig{}, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	return isf.config, nil
}

func (*InputManager) SetInputConfig(
	e common.EntityId,
	config InputConfig,
	world *World,
) error {
	isf, err := world.Inputs.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	isf.config = config

	return nil
}

func (*InputManager) GetInputSourceFunc(
	e common.EntityId,
	world *World,
) (InputSourceFunc, error) {
	isf, err := world.Inputs.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	return isf.inputSourceFunc, nil
}

func (*InputManager) SetInputSourceFunc(
	e common.EntityId,
	inputSourceFunc InputSourceFunc,
	world *World,
) error {
	isf, err := world.Inputs.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	isf.inputSourceFunc = inputSourceFunc

	return nil
}
