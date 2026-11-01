package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
	"slices"
)

type InputLoopParams struct {
	LoopInputs  []ecs.InputState
	CurrentTick uint64
}

func (InputLoopParams) IsInputParams() {}

func (p *InputLoopParams) Copy() ecs.InputParams {
	if p == nil {
		return p
	}
	c := *p
	c.LoopInputs = slices.Clone(p.LoopInputs)
	return &c
}

func LoopInputSource(
	entityId common.EntityId,
	tick uint64,
	params ecs.InputParams,
	ecsContainer *ecs.ECSContainer,
) (ecs.InputState, error) {
	if params == nil {
		return ecs.InputState{}, fmt.Errorf("input params are nil")
	}

	loopParams, ok := params.(*InputLoopParams)
	if !ok {
		return ecs.InputState{}, fmt.Errorf("input params are not of type InputLoopParams")
	}
	if loopParams == nil {
		return ecs.InputState{}, fmt.Errorf("input params are nil")
	}

	if len(loopParams.LoopInputs) == 0 {
		return ecs.InputState{}, fmt.Errorf("loopInputs is empty")
	}
	loopParams.CurrentTick++
	if loopParams.CurrentTick >= uint64(len(loopParams.LoopInputs)) {
		loopParams.CurrentTick = 0
	}
	return loopParams.LoopInputs[loopParams.CurrentTick], nil
}
