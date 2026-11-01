package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
	"maps"
)

type InputDemoParams struct {
	Log map[uint64]map[common.EntityId]ecs.InputState
}

func (InputDemoParams) IsInputParams() {}

func (p *InputDemoParams) Copy() ecs.InputParams {
	if p == nil {
		return p
	}
	return &InputDemoParams{Log: copyInputLog(p.Log)}
}

func copyInputLog(inputLog map[uint64]map[common.EntityId]ecs.InputState) map[uint64]map[common.EntityId]ecs.InputState {
	if inputLog == nil {
		return nil
	}

	c := make(map[uint64]map[common.EntityId]ecs.InputState, len(inputLog))
	for tick, inputs := range inputLog {
		c[tick] = maps.Clone(inputs)
	}

	return c
}

func DemoInputSource(
	entityId common.EntityId,
	tick uint64,
	params ecs.InputParams,
	ecsContainer *ecs.ECSContainer,
) (ecs.InputState, error) {
	if params == nil {
		return ecs.InputState{}, fmt.Errorf("input params are nil")
	}

	demoParams, ok := params.(*InputDemoParams)
	if !ok {
		return ecs.InputState{}, fmt.Errorf("input params are not of type InputDemoParams")
	}
	if demoParams == nil {
		return ecs.InputState{}, fmt.Errorf("input params are nil")
	}

	return demoParams.Log[tick][entityId], nil
}
