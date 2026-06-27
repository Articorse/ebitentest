package ecs

import (
	"ebittest/ecs/common"
)

func DummyInputSource(
	entityId common.EntityId,
	tick uint64,
	params InputParams,
	ecsContainer *ECSContainer,
) (InputState, error) {
	return InputState{}, nil
}
