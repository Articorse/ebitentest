package timersystem

import (
	"ebittest/ecs"
	"fmt"
	"slices"
)

func Tick(ecs *ecs.ECS) error {
	tm := ecs.TimerManager
	for _, e := range slices.Clone(ecs.Timers.GetEntities()) {
		_, err := tm.TickDown(e, ecs)
		if err != nil {
			return fmt.Errorf("error ticking down timer component of entity %d: %v", e, err)
		}
	}

	return nil
}
