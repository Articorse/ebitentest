package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func DummyInputSource(
	entityId common.EntityId,
	tick uint64,
	world *ecs.World,
) ecs.InputState {
	return ecs.InputState{}
}
