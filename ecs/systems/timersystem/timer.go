package timersystem

import (
	"ebittest/ecs"
	"fmt"
)

func Tick(world *ecs.World) error {
	tm := ecs.TimerManager{}
	for e, _ := range world.Timers {
		_, err := tm.TickDown(e, world)
		if err != nil {
			return fmt.Errorf("error ticking down timed life component of entity %d: %v", e, err)
		}
	}

	return nil
}
