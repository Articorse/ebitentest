package components

import (
	"ebittest/data"
	"ebittest/ecs/ecscommon"
	"fmt"
)

type TimedLifeManager struct{}

func NewTimedLifeComponent(duration int64) *TimedLife {
	return &TimedLife{remainingMs: duration}
}

func (*TimedLifeManager) GetRemainingMs(
	e ecscommon.EntityId,
	timedlives map[ecscommon.EntityId]*TimedLife,
) (int64, error) {
	timedLifeComp, ok := timedlives[e]
	if !ok {
		return 0, fmt.Errorf("could not get timed life component of entity %d", e)
	}

	return timedLifeComp.remainingMs, nil
}

func (*TimedLifeManager) TickDown(
	e ecscommon.EntityId,
	timedlives map[ecscommon.EntityId]*TimedLife,
) (timerOver bool, err error) {
	timedLifeComp, ok := timedlives[e]
	if !ok {
		return false, fmt.Errorf("could not get timed life component of entity %d", e)
	}

	timedLifeComp.remainingMs -= 1000 / data.TPS

	if timedLifeComp.remainingMs <= 0 {
		return true, nil
	}

	return false, nil
}
