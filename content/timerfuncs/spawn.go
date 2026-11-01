package timerfuncs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
)

func Timer_Spawn(self common.EntityId, ecsContainer *ecs.ECSContainer) error {
	_, err := ecsContainer.SpawnerManager.Spawn(self, ecsContainer)
	if err != nil {
		return fmt.Errorf("error during enemy spawn: %v", err)
	}
	return nil
}
