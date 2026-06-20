package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func NewReplayInputSource(
	replayStartTick uint64,
	replayInputs map[uint64]ecs.InputState,
) ecs.InputSourceFunc {
	return func(
		entityId common.EntityId,
		tick uint64,
		ecs *ecs.ECS,
	) ecs.InputState {
		relTick := tick - replayStartTick
		relTickInput, ok := replayInputs[relTick]
		if !ok {
			return ecs.InputState{}
		}
		return relTickInput
	}
}
