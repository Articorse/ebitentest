package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"log"
)

func NewFollowInputSource(followEntity common.EntityId) ecs.InputSourceFunc {
	return func(
		entityId common.EntityId,
		tick uint64,
		ecs *ecs.ECS,
	) ecs.InputState {
		tm := ecs.TransformManager
		is := ecs.InputState{}

		selfWorldPos, err := tm.GetWorldPos(entityId, ecs)
		if err != nil {
			log.Printf("error getting ecs position for self entity %d: %v\n", entityId, err)
			return is
		}

		targetWorldPos, err := tm.GetWorldPos(followEntity, ecs)
		if err != nil {
			log.Printf("error getting ecs position for follow entity %d: %v\n", followEntity, err)
			return is
		}

		dir := targetWorldPos.Subtract(selfWorldPos).Normalized()

		is.Analog1X = dir.X
		is.Analog1Y = dir.Y

		return is
	}
}
