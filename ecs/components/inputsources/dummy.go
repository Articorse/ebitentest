package inputsources

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
)

func DummyInputSource(
	entityId ecscommon.EntityId,
	tick uint64,
	inputs map[ecscommon.EntityId]*components.Input,
) components.InputState {
	return components.InputState{}
}
