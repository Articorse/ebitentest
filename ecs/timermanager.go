package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
)

type timerManager struct{}

// TODO: Fix triggerCount = 0 being infinite triggers
func NewTimerComponent(
	counterMs int,
	triggerCount int, // Set to -1 to repeat infinitely
	timerFunc TimerFunc,
) (*timer, error) {
	if timerFunc == nil {
		return nil, fmt.Errorf("timer function cannot be nil")
	}

	return &timer{
		counterMs:         counterMs,
		maxTimeMs:         counterMs,
		remainingTriggers: triggerCount,
		timerFunc:         timerFunc,
	}, nil
}

func (timerManager) TickDown(
	e common.EntityId,
	world *World,
) (timerOver bool, err error) {
	timer, err := world.Timers.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get timer component of entity %d: %v", e, err)
	}

	if timer.remainingTriggers == 0 {
		return true, nil
	}

	timer.counterMs -= data.TickMs

	if timer.counterMs <= 0 {
		err := timer.timerFunc(e, world)
		timer.remainingTriggers--
		timer.counterMs = timer.maxTimeMs
		if err != nil {
			return false, fmt.Errorf("error executing timer function of entity %d: %v", e, err)
		}

		return true, nil
	}

	return false, nil
}
