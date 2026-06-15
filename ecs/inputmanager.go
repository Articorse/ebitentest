package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type InputManager struct{}

type FacingInputEnum uint8

const (
	Facing_None FacingInputEnum = iota
	Facing_Mouse
	Facing_Analog
)

type InputState struct {
	Analog1X, Analog1Y float64
	Analog2X, Analog2Y float64
	Ability1           float64
	Ability2           float64
	FacingDir          utils.Vec2
}

type InputSourceFunc func(
	e common.EntityId,
	tick uint64,
	world *World,
) InputState

func NewInputComponent(config map[InputType]InputKey, inputSourceFunc InputSourceFunc, facingInput FacingInputEnum) *input {
	return &input{config: config, inputSourceFunc: inputSourceFunc, facingInput: facingInput}
}

func (*InputManager) GetInput(e common.EntityId, inputType InputType, world *World) (float64, error) {
	inComp, err := world.Inputs.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	inKey, ok := inComp.config[inputType]
	if !ok {
		return 0, nil
	}

	return inKey.GetInput(), nil
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

func (*InputManager) GetFacingInput(e common.EntityId, world *World) (FacingInputEnum, error) {
	inComp, err := world.Inputs.getComponent(e)
	if err != nil {
		return Facing_None, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	return inComp.facingInput, nil
}
