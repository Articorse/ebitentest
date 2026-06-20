package ecs

import "ebittest/ecs/common"

type TimerFunc func(self common.EntityId, ecs *ECS) error

type timer struct {
	counterMs         int
	maxTimeMs         int
	remainingTriggers int
	timerFunc         TimerFunc
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
