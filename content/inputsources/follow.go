package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
)

type InputFollowParams struct {
	FollowEntity common.EntityId
}

func (InputFollowParams) IsInputParams() {}

func (p *InputFollowParams) Copy() ecs.InputParams {
	if p == nil {
		return p
	}
	c := *p
	return &c
}

func FollowInputSource(
	entityId common.EntityId,
	tick uint64,
	params ecs.InputParams,
	ecsContainer *ecs.ECSContainer,
) (ecs.InputState, error) {
	if params == nil {
		return ecs.InputState{}, fmt.Errorf("input params are nil")
	}

	followParams, ok := params.(*InputFollowParams)
	if !ok {
		return ecs.InputState{}, fmt.Errorf("input params are not of type InputFollowParams")
	}
	if followParams == nil {
		return ecs.InputState{}, fmt.Errorf("input params are nil")
	}

	is := ecs.InputState{}

	selfWorldPos, err := ecsContainer.WorldPos(entityId)
	if err != nil {
		return is, fmt.Errorf("error getting world position for self entity %d: %v", entityId, err)
	}

	targetWorldPos, err := ecsContainer.WorldPos(followParams.FollowEntity)
	if err != nil {
		return is, fmt.Errorf("error getting world position for follow entity %d: %v", followParams.FollowEntity, err)
	}

	dir := targetWorldPos.Subtract(selfWorldPos).Normalized()

	is.Analog1X = dir.X
	is.Analog1Y = dir.Y

	return is, nil
}
