package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func DummyInputSource(
	entityId common.EntityId,
	tick uint64,
	params ecs.InputParams,
	ecsContainer *ecs.ECSContainer,
) (ecs.InputState, error) {
	return ecs.InputState{}, nil
}
