package timersystem

import (
	"ebittest/ecs"
	"fmt"
	"slices"
)

func Tick(ecsContainer *ecs.ECSContainer) error {
	tm := ecsContainer.TimerManager
	for _, e := range slices.Clone(ecsContainer.Timers.GetEntities()) {
		_, err := tm.TickDown(e, ecsContainer)
		if err != nil {
			return fmt.Errorf("error ticking down timer component of entity %d: %v", e, err)
		}
	}

	return nil
}
