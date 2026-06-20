package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type inputManager struct{}

type FacingInputEnum uint8

const (
	Facing_None FacingInputEnum = iota
	Facing_Mouse
	Facing_Analog2
)

type InputState struct {
	Analog1X, Analog1Y float64
	Analog2X, Analog2Y float64
	MainHandEqAbility1 float64
	MainHandEqAbility2 float64
	OffHandEqAbility1  float64
	OffHandEqAbility2  float64
	Ability1           float64
	Ability2           float64
	FacingDir          utils.Vec2
}

type InputSourceFunc func(
	e common.EntityId,
	tick uint64,
	ecs *ECS,
) InputState

func NewInputComponent(config map[InputType]InputKey, inputSourceFunc InputSourceFunc, facingInput FacingInputEnum) *input {
	return &input{config: config, inputSourceFunc: inputSourceFunc, facingInput: facingInput}
}

func (*inputManager) GetInput(e common.EntityId, inputType InputType, ecs *ECS) (float64, error) {
	inComp, err := ecs.Inputs.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	inKey, ok := inComp.config[inputType]
	if !ok {
		return 0, nil
	}

	return inKey.GetInput(), nil
}

func (*inputManager) GetInputSourceFunc(
	e common.EntityId,
	ecs *ECS,
) (InputSourceFunc, error) {
	isf, err := ecs.Inputs.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	return isf.inputSourceFunc, nil
}

func (*inputManager) SetInputSourceFunc(
	e common.EntityId,
	inputSourceFunc InputSourceFunc,
	ecs *ECS,
) error {
	isf, err := ecs.Inputs.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	isf.inputSourceFunc = inputSourceFunc

	return nil
}

func (*inputManager) GetFacingInput(e common.EntityId, ecs *ECS) (FacingInputEnum, error) {
	inComp, err := ecs.Inputs.getComponent(e)
	if err != nil {
		return Facing_None, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	return inComp.facingInput, nil
}

func (*inputManager) GetLastFacingDir(e common.EntityId, ecs *ECS) (utils.Vec2, error) {
	inComp, err := ecs.Inputs.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	return inComp.lastFacingDir, nil
}

func (*inputManager) SetLastFacingDir(e common.EntityId, facingDir utils.Vec2, ecs *ECS) error {
	inComp, err := ecs.Inputs.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	inComp.lastFacingDir = facingDir

	return nil
}
