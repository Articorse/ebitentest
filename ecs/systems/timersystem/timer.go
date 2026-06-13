package timersystem

import (
	"ebittest/ecs"
	"fmt"
	"slices"
)

func Tick(world *ecs.World) error {
	tm := ecs.TimerManager{}
	for _, e := range slices.Clone(world.Timers.GetEntities()) {
		_, err := tm.TickDown(e, world)
		if err != nil {
			return fmt.Errorf("error ticking down timer component of entity %d: %v", e, err)
		}
	}

	return nil
}
