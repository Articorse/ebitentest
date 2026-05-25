package inputsources

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
)

func NewReplayInputSource(
	replayStartTick uint64,
	replayInputs map[uint64]components.InputState,
) components.InputSourceFunc {
	return func(
		entityId ecscommon.EntityId,
		tick uint64,
		inputs map[ecscommon.EntityId]*components.Input,
	) components.InputState {
		relTick := tick - replayStartTick
		relTickInput, ok := replayInputs[relTick]
		if !ok {
			return components.InputState{}
		}
		return relTickInput
	}
}
