package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func DummyInputSource(
	entityId common.EntityId,
	tick uint64,
	ecs *ecs.ECS,
) ecs.InputState {
	return ecs.InputState{}
}
