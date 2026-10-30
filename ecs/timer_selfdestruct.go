package ecs

import "ebittest/ecs/common"

func Timer_Selfdestruct(self common.EntityId, ecsContainer *ECSContainer) error {
	ecsContainer.ScheduleKillEntity(self)
	return nil
}
