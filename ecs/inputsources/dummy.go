package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func DummyInputSource(
	entityId common.EntityId,
	tick uint64,
	ecsContainer *ecs.ECSContainer,
) ecs.InputState {
	return ecs.InputState{}
}
