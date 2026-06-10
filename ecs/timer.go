package ecs

import "ebittest/ecs/common"

type TimerFunc func(self common.EntityId, world *World) error

type timer struct {
	counterMs        int
	maxTimeMs        int
	remainingRepeats int
	timerFunc        TimerFunc
}

func (timer) isComponent() {}

func (x timer) Copy() timer {
	return timer{
		counterMs:        x.counterMs,
		maxTimeMs:        x.maxTimeMs,
		remainingRepeats: x.remainingRepeats,
		timerFunc:        x.timerFunc,
	}
}
