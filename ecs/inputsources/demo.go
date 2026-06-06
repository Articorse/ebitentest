package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func DemoInputSource(
	log map[uint64]map[common.EntityId]ecs.InputState,
) ecs.InputSourceFunc {
	return func(entityId common.EntityId, tick uint64, world *ecs.World) ecs.InputState {
		return log[tick][entityId]
	}
}
