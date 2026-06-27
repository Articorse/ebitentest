package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type InputReplayParams struct {
	ReplayStartTick uint64
	ReplayInputs    map[uint64]InputState
}

func (InputReplayParams) isInputParams() {}

func ReplayInputSource(
	entityId common.EntityId,
	tick uint64,
	params InputParams,
	ecsContainer *ECSContainer,
) (InputState, error) {
	if params == nil {
		return InputState{}, fmt.Errorf("input params are nil")
	}

	var replayParams InputReplayParams
	switch p := params.(type) {
	case InputReplayParams:
		replayParams = p
	case *InputReplayParams:
		if p == nil {
			return InputState{}, fmt.Errorf("input params are nil")
		}
		replayParams = *p
	default:
		return InputState{}, fmt.Errorf("input params are not of type InputReplayParams")
	}

	relTick := tick - replayParams.ReplayStartTick
	relTickInput, ok := replayParams.ReplayInputs[relTick]
	if !ok {
		return InputState{}, fmt.Errorf("no input found for relative tick %d", relTick)
	}
	return relTickInput, nil
}
