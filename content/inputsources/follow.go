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

	return towards(entityId, followParams.FollowEntity, ecsContainer)
}

// Heads straight at the target, whatever's in the way
func towards(self, target common.EntityId, ecsContainer *ecs.ECSContainer) (ecs.InputState, error) {
	is := ecs.InputState{}

	selfWorldPos, err := ecsContainer.WorldPos(self)
	if err != nil {
		return is, fmt.Errorf("error getting world position for self entity %d: %v", self, err)
	}

	targetWorldPos, err := ecsContainer.WorldPos(target)
	if err != nil {
		return is, fmt.Errorf("error getting world position for follow entity %d: %v", target, err)
	}

	dir := targetWorldPos.Subtract(selfWorldPos).Normalized()

	is.Analog1X = dir.X
	is.Analog1Y = dir.Y

	return is, nil
}
