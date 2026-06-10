package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
)

type TimerManager struct{}

func NewTimerComponent(
	counterMs int,
	repeatCount int, // Set to -1 to repeat infinitely
	timerFunc TimerFunc,
) (*timer, error) {
	if timerFunc == nil {
		return nil, fmt.Errorf("timer function cannot be nil")
	}

	return &timer{
		counterMs:        counterMs,
		maxTimeMs:        counterMs,
		remainingRepeats: repeatCount,
		timerFunc:        timerFunc,
	}, nil
}

func (TimerManager) TickDown(
	e common.EntityId,
	world *World,
) (timerOver bool, err error) {
	timer, ok := world.Timers.getData()[e]
	if !ok {
		return false, fmt.Errorf("could not get timer component of entity %d", e)
	}

	if timer.remainingRepeats == 0 {
		return true, nil
	}

	timer.counterMs -= 1000 / data.TPS

	if timer.counterMs <= 0 {
		err := timer.timerFunc(e, world)
		timer.remainingRepeats--
		timer.counterMs = timer.maxTimeMs
		if err != nil {
			return false, fmt.Errorf("error executing timer function of entity %d: %v", e, err)
		}

		return true, nil
	}

	return false, nil
}
