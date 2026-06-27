package ecs

import "ebittest/ecs/common"

type TimerFuncEnum uint64

const (
	TimerFunc_None TimerFuncEnum = iota
	TimerFunc_Selfdestruct
	TimerFunc_Spawn
)

type TimerFunc func(self common.EntityId, ecsContainer *ECSContainer) error

type timer struct {
	counterMs         int
	maxTimeMs         int
	remainingTriggers int
	timerFunc         TimerFuncEnum
}

func (timer) isComponent() {}

func (x timer) Copy() timer {
	return timer{
		counterMs:         x.counterMs,
		maxTimeMs:         x.maxTimeMs,
		remainingTriggers: x.remainingTriggers,
		timerFunc:         x.timerFunc,
	}
}

type timerDto struct {
	CounterMs         int
	MaxTimeMs         int
	RemainingTriggers int
	TimerFunc         TimerFuncEnum
}

func (timerDto) isComponentDto() {}

func (x timer) ToDto() timerDto {
	return timerDto{
		CounterMs:         x.counterMs,
		MaxTimeMs:         x.maxTimeMs,
		RemainingTriggers: x.remainingTriggers,
		TimerFunc:         x.timerFunc,
	}
}

func (x timerDto) ToComponent() *timer {
	return &timer{
		counterMs:         x.CounterMs,
		maxTimeMs:         x.MaxTimeMs,
		remainingTriggers: x.RemainingTriggers,
		timerFunc:         x.TimerFunc,
	}
}
