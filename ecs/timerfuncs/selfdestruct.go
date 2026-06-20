package timerfuncs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func Selfdestruct(self common.EntityId, ecs *ecs.ECS) error {
	ecs.ScheduleRemoveEntity(self)
	return nil
}
