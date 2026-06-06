package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
)

type TimedLifeManager struct{}

func NewTimedLifeComponent(duration int64) *timedLife {
	return &timedLife{remainingMs: duration}
}

func (*TimedLifeManager) GetRemainingMs(
	e common.EntityId,
	timedlives map[common.EntityId]*timedLife,
) (int64, error) {
	timedLifeComp, ok := timedlives[e]
	if !ok {
		return 0, fmt.Errorf("could not get timed life component of entity %d", e)
	}

	return timedLifeComp.remainingMs, nil
}

func (*TimedLifeManager) TickDown(
	e common.EntityId,
	timedlives map[common.EntityId]*timedLife,
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
