package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func NewLoopInputSource(loopInputs []ecs.InputState, startTick uint64) ecs.InputSourceFunc {
	return func(
		entityId common.EntityId,
		tick uint64,
		ecsContainer *ecs.ECSContainer,
	) ecs.InputState {
		if len(loopInputs) == 0 {
			return ecs.InputState{}
		}
		idx := (tick - startTick) % uint64(len(loopInputs))
		return loopInputs[idx]
	}
}
