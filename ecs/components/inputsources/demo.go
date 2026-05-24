package inputsources

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
)

func DemoInputSource(
	log map[uint64]map[ecscommon.EntityId]components.InputState,
	inputs map[ecscommon.EntityId]*components.Input,
) components.InputSourceFunc {
	return func(entityId ecscommon.EntityId, tick uint64, inputs map[ecscommon.EntityId]*components.Input) components.InputState {
		return log[tick][entityId]
	}
}
