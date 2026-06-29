package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type InputLoopParams struct {
	LoopInputs  []InputState
	CurrentTick uint64
}

func (InputLoopParams) isInputParams() {}

func LoopInputSource(
	entityId common.EntityId,
	tick uint64,
	params InputParams,
	ecsContainer *ECSContainer,
) (InputState, error) {
	if params == nil {
		return InputState{}, fmt.Errorf("input params are nil")
	}

	var loopParams *InputLoopParams
	switch p := params.(type) {
	case InputLoopParams:
		loopParams = &p
	case *InputLoopParams:
		if p == nil {
			return InputState{}, fmt.Errorf("input params are nil")
		}
		loopParams = p
	default:
		return InputState{}, fmt.Errorf("input params are not of type InputLoopParams")
	}

	if len(loopParams.LoopInputs) == 0 {
		return InputState{}, fmt.Errorf("loopInputs is empty")
	}
	loopParams.CurrentTick++
	if loopParams.CurrentTick >= uint64(len(loopParams.LoopInputs)) {
		loopParams.CurrentTick = 0
	}
	return loopParams.LoopInputs[loopParams.CurrentTick], nil
}
