package timerfuncs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
)

func Spawn(self common.EntityId, ecs *ecs.ECS) error {
	_, err := ecs.SpawnerManager.Spawn(self, ecs)
	if err != nil {
		return fmt.Errorf("error during enemy spawn: %v", err)
	}
	return nil
}
