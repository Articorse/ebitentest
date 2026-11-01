package timerfuncs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
)

func Timer_Selfdestruct(self common.EntityId, ecsContainer *ecs.ECSContainer) error {
	ecsContainer.ScheduleKillEntity(self)
	return nil
}
