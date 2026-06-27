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

func NewInputComponent(config map[InputType]InputKey, inputType InputTypeEnum, params InputParams, facingInput FacingInputEnum) *input {
	return &input{config: config, inputType: inputType, params: params, facingInput: facingInput}
}

func (*inputManager) GetInput(e common.EntityId, inputType InputType, ecsContainer *ECSContainer) (float64, error) {
	inComp, err := ecsContainer.Inputs.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	inKey, ok := inComp.config[inputType]
	if !ok {
		return 0, nil
	}

	return inKey.GetInput(), nil
}

func (*inputManager) GetParams(e common.EntityId, ecsContainer *ECSContainer) (InputParams, error) {
	inComp, err := ecsContainer.Inputs.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	return inComp.params, nil
}

func (*inputManager) GetInputType(e common.EntityId, ecsContainer *ECSContainer) (InputTypeEnum, error) {
	inComp, err := ecsContainer.Inputs.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	return inComp.inputType, nil
}

func (*inputManager) GetFacingInput(e common.EntityId, ecsContainer *ECSContainer) (FacingInputEnum, error) {
	inComp, err := ecsContainer.Inputs.getComponent(e)
	if err != nil {
		return Facing_None, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	return inComp.facingInput, nil
}

func (*inputManager) GetLastFacingDir(e common.EntityId, ecsContainer *ECSContainer) (utils.Vec2, error) {
	inComp, err := ecsContainer.Inputs.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	return inComp.lastFacingDir, nil
}

func (*inputManager) SetLastFacingDir(e common.EntityId, facingDir utils.Vec2, ecsContainer *ECSContainer) error {
	inComp, err := ecsContainer.Inputs.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get input of entity %d: %v", e, err)
	}

	inComp.lastFacingDir = facingDir

	return nil
}
