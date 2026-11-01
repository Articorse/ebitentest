package timerfuncs

import (
	"ebittest/ecs"
	"fmt"
)

// Saved components refer to timer functions by these values, so they must never change
const (
	TimerFunc_Selfdestruct ecs.TimerFuncEnum = 1
	TimerFunc_Spawn        ecs.TimerFuncEnum = 2
)

func Register(ecsContainer *ecs.ECSContainer) error {
	timerFuncs := []struct {
		id        ecs.TimerFuncEnum
		timerFunc ecs.TimerFunc
	}{
		{TimerFunc_Selfdestruct, Timer_Selfdestruct},
		{TimerFunc_Spawn, Timer_Spawn},
	}

	for _, t := range timerFuncs {
		if err := ecsContainer.RegisterTimerFunc(t.id, t.timerFunc); err != nil {
			return fmt.Errorf("error registering timer function %v: %w", t.id, err)
		}
	}

	return nil
}
