package timerfuncs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func Selfdestruct(self common.EntityId, world *ecs.World) error {
	world.ScheduleRemoveEntity(self)
	return nil
}
