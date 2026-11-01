package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
)

type TimerFuncEnum uint64

// The other IDs are defined by the content that registers them
const TimerFunc_None TimerFuncEnum = 0

type TimerFunc func(self common.EntityId, ecsContainer *ECSContainer) error

type Timer struct {
	counterMs         int
	maxTimeMs         int
	remainingTriggers int
	timerFunc         TimerFuncEnum
}

// TODO: Fix triggerCount = 0 being infinite triggers
func NewTimerComponent(
	counterMs int,
	triggerCount int, // Set to -1 to repeat infinitely
	timerFunc TimerFuncEnum,
) (*Timer, error) {
	if timerFunc == TimerFunc_None {
		return nil, fmt.Errorf("timer function cannot be none")
	}

	return &Timer{
		counterMs:         counterMs,
		maxTimeMs:         counterMs,
		remainingTriggers: triggerCount,
		timerFunc:         timerFunc,
	}, nil
}

func (x *ECSContainer) TickTimer(e common.EntityId) (timerOver bool, err error) {
	t, ok := x.Timers.Get(e)
	if !ok {
		return false, fmt.Errorf("entity %d has no timer", e)
	}

	if t.remainingTriggers == 0 {
		return true, nil
	}

	t.counterMs -= data.TickMs

	if t.counterMs <= 0 {
		timerFunc, err := x.TimerFunc(t.timerFunc)
		if err != nil {
			return false, fmt.Errorf("could not get timer function of entity %d: %v", e, err)
		}
		if err := timerFunc(e, x); err != nil {
			return false, fmt.Errorf("error executing timer function of entity %d: %v", e, err)
		}
		t.remainingTriggers--
		t.counterMs = t.maxTimeMs

		return true, nil
	}

	return false, nil
}

func (x *ECSContainer) TimerFunc(timerFuncId TimerFuncEnum) (TimerFunc, error) {
	if timerFuncId == TimerFunc_None {
		return nil, nil
	}

	timerFunc, ok := x.timerFuncs[timerFuncId]
	if !ok {
		return nil, fmt.Errorf("timer function for timer %v not found", timerFuncId)
	}

	return timerFunc, nil
}

func (x Timer) Copy() Timer {
	return Timer{
		counterMs:         x.counterMs,
		maxTimeMs:         x.maxTimeMs,
		remainingTriggers: x.remainingTriggers,
		timerFunc:         x.timerFunc,
	}
}

func (x Timer) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Timers.addComponent(e, x.Copy())
}

type timerDto struct {
	CounterMs         int
	MaxTimeMs         int
	RemainingTriggers int
	TimerFunc         TimerFuncEnum
}

func (x Timer) toDto() (ComponentDto, error) {
	return &timerDto{
		CounterMs:         x.counterMs,
		MaxTimeMs:         x.maxTimeMs,
		RemainingTriggers: x.remainingTriggers,
		TimerFunc:         x.timerFunc,
	}, nil
}

func (x timerDto) toComponent() (Component, error) {
	return &Timer{
		counterMs:         x.CounterMs,
		maxTimeMs:         x.MaxTimeMs,
		remainingTriggers: x.RemainingTriggers,
		timerFunc:         x.TimerFunc,
	}, nil
}
