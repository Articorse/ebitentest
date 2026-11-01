package ecs

import "ebittest/ecs/common"

type TimerFuncEnum uint64

// The other IDs are defined by the content that registers them
const TimerFunc_None TimerFuncEnum = 0

type TimerFunc func(self common.EntityId, ecsContainer *ECSContainer) error

type timer struct {
	counterMs         int
	maxTimeMs         int
	remainingTriggers int
	timerFunc         TimerFuncEnum
}

func (x timer) Copy() timer {
	return timer{
		counterMs:         x.counterMs,
		maxTimeMs:         x.maxTimeMs,
		remainingTriggers: x.remainingTriggers,
		timerFunc:         x.timerFunc,
	}
}

func (x timer) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Timers.addComponent(e, x.Copy())
}

type timerDto struct {
	CounterMs         int
	MaxTimeMs         int
	RemainingTriggers int
	TimerFunc         TimerFuncEnum
}

func (x timer) toDto() (ComponentDto, error) {
	return &timerDto{
		CounterMs:         x.counterMs,
		MaxTimeMs:         x.maxTimeMs,
		RemainingTriggers: x.remainingTriggers,
		TimerFunc:         x.timerFunc,
	}, nil
}

func (x timerDto) toComponent() (Component, error) {
	return &timer{
		counterMs:         x.CounterMs,
		maxTimeMs:         x.MaxTimeMs,
		remainingTriggers: x.RemainingTriggers,
		timerFunc:         x.TimerFunc,
	}, nil
}
