package inputsources

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
)

func NewLoopInputSource(loopInputs []components.InputState, startTick uint64) components.InputSourceFunc {
	return func(
		entityId ecscommon.EntityId,
		tick uint64,
		inputs map[ecscommon.EntityId]*components.Input,
	) components.InputState {
		if len(loopInputs) == 0 {
			return components.InputState{}
		}
		idx := (tick - startTick) % uint64(len(loopInputs))
		return loopInputs[idx]
	}
}
