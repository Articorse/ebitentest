package timerfuncs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
)

func Selfdestruct(self common.EntityId, world *ecs.World) error {
	err := world.RemoveEntity(self)
	if err != nil {
		return fmt.Errorf("error self-destructing entity %d: %v", self, err)
	}
	return nil
}
