package timerfuncs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func Selfdestruct(self common.EntityId, ecsContainer *ecs.ECSContainer) error {
	ecsContainer.ScheduleRemoveEntity(self)
	return nil
}
