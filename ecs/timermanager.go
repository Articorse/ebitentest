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
	timerFunc TimerFuncEnum,
) (*timer, error) {
	if timerFunc == TimerFunc_None {
		return nil, fmt.Errorf("timer function cannot be none")
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
	ecsContainer *ECSContainer,
) (timerOver bool, err error) {
	timer, err := ecsContainer.Timers.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get timer component of entity %d: %v", e, err)
	}

	if timer.remainingTriggers == 0 {
		return true, nil
	}

	timer.counterMs -= data.TickMs

	if timer.counterMs <= 0 {
		timerFunc, err := ecsContainer.TimerManager.GetTimerFunc(timer.timerFunc)
		if err != nil {
			return false, fmt.Errorf("could not get timer function of entity %d: %v", e, err)
		}
		err = timerFunc(e, ecsContainer)
		if err != nil {
			return false, fmt.Errorf("error executing timer function of entity %d: %v", e, err)
		}
		timer.remainingTriggers--
		timer.counterMs = timer.maxTimeMs
		if err != nil {
			return false, fmt.Errorf("error executing timer function of entity %d: %v", e, err)
		}

		return true, nil
	}

	return false, nil
}

func (timerManager) GetTimerFunc(
	timerFuncId TimerFuncEnum,
) (TimerFunc, error) {
	switch timerFuncId {
	case TimerFunc_None:
		return nil, nil
	case TimerFunc_Selfdestruct:
		return Timer_Selfdestruct, nil
	case TimerFunc_Spawn:
		return Timer_Spawn, nil
	default:
		return nil, fmt.Errorf("timer function for timer %v not found", timerFuncId)
	}
}
