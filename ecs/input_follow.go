package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type InputFollowParams struct {
	FollowEntity common.EntityId
}

func (InputFollowParams) isInputParams() {}

func FollowInputSource(
	entityId common.EntityId,
	tick uint64,
	params InputParams,
	ecsContainer *ECSContainer,
) (InputState, error) {
	if params == nil {
		return InputState{}, fmt.Errorf("input params are nil")
	}

	var followParams *InputFollowParams
	switch p := params.(type) {
	case InputFollowParams:
		followParams = &p
	case *InputFollowParams:
		if p == nil {
			return InputState{}, fmt.Errorf("input params are nil")
		}
		followParams = p
	default:
		return InputState{}, fmt.Errorf("input params are not of type InputFollowParams")
	}

	tm := ecsContainer.TransformManager
	is := InputState{}

	selfWorldPos, err := tm.GetWorldPos(entityId, ecsContainer)
	if err != nil {
		return is, fmt.Errorf("error getting world position for self entity %d: %v", entityId, err)
	}

	targetWorldPos, err := tm.GetWorldPos(followParams.FollowEntity, ecsContainer)
	if err != nil {
		return is, fmt.Errorf("error getting world position for follow entity %d: %v", followParams.FollowEntity, err)
	}

	dir := targetWorldPos.Subtract(selfWorldPos).Normalized()

	is.Analog1X = dir.X
	is.Analog1Y = dir.Y

	return is, nil
}
