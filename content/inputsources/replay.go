package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
	"maps"
)

type InputReplayParams struct {
	ReplayStartTick uint64
	ReplayInputs    map[uint64]ecs.InputState
}

func (InputReplayParams) IsInputParams() {}

func (p *InputReplayParams) Copy() ecs.InputParams {
	if p == nil {
		return p
	}
	c := *p
	c.ReplayInputs = maps.Clone(p.ReplayInputs)
	return &c
}

func ReplayInputSource(
	entityId common.EntityId,
	tick uint64,
	params ecs.InputParams,
	ecsContainer *ecs.ECSContainer,
) (ecs.InputState, error) {
	if params == nil {
		return ecs.InputState{}, fmt.Errorf("input params are nil")
	}

	replayParams, ok := params.(*InputReplayParams)
	if !ok {
		return ecs.InputState{}, fmt.Errorf("input params are not of type InputReplayParams")
	}
	if replayParams == nil {
		return ecs.InputState{}, fmt.Errorf("input params are nil")
	}

	relTick := tick - replayParams.ReplayStartTick
	relTickInput, ok := replayParams.ReplayInputs[relTick]
	if !ok {
		return ecs.InputState{}, fmt.Errorf("no input found for relative tick %d", relTick)
	}
	return relTickInput, nil
}
