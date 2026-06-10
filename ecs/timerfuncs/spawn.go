package timerfuncs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
)

func Spawn(self common.EntityId, world *ecs.World) error {
	sm := ecs.SpawnerManager{}
	err := sm.Spawn(self, world)
	if err != nil {
		return fmt.Errorf("error during enemy spawn: %v", err)
	}
	return nil
}
