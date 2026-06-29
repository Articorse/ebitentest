package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type InputDemoParams struct {
	Log map[uint64]map[common.EntityId]InputState
}

func (InputDemoParams) isInputParams() {}

func DemoInputSource(
	entityId common.EntityId,
	tick uint64,
	params InputParams,
	ecsContainer *ECSContainer,
) (InputState, error) {
	if params == nil {
		return InputState{}, fmt.Errorf("input params are nil")
	}

	var demoParams *InputDemoParams
	switch p := params.(type) {
	case InputDemoParams:
		demoParams = &p
	case *InputDemoParams:
		if p == nil {
			return InputState{}, fmt.Errorf("input params are nil")
		}
		demoParams = p
	default:
		return InputState{}, fmt.Errorf("input params are not of type InputDemoParams")
	}

	return demoParams.Log[tick][entityId], nil
}
