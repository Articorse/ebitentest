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
		world *ecs.World,
	) ecs.InputState {
		tm := world.TransformManager
		is := ecs.InputState{}

		selfWorldPos, err := tm.GetWorldPos(entityId, world)
		if err != nil {
			log.Printf("error getting world position for self entity %d: %v\n", entityId, err)
			return is
		}

		targetWorldPos, err := tm.GetWorldPos(followEntity, world)
		if err != nil {
			log.Printf("error getting world position for follow entity %d: %v\n", followEntity, err)
			return is
		}

		dir := targetWorldPos.Subtract(selfWorldPos).Normalized()

		is.Analog1X = dir.X
		is.Analog1Y = dir.Y

		return is
	}
}
