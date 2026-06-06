package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type HitpointsManager struct{}

func NewHitpointsComponent(max int64) *hitpoints {
	return &hitpoints{
		max:     max,
		current: max,
	}
}

func (HitpointsManager) GetMax(
	e common.EntityId,
	hitpoints map[common.EntityId]*hitpoints,
) (int64, error) {
	hpComp, ok := hitpoints[e]
	if !ok {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d", e)
	}

	return hpComp.max, nil
}

// Returns -1 if component not found
func (HitpointsManager) GetCurrent(
	e common.EntityId,
	hitpoints map[common.EntityId]*hitpoints,
) (int64, error) {
	hpComp, ok := hitpoints[e]
	if !ok {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d", e)
	}

	return hpComp.current, nil
}

func (HitpointsManager) TakeDamage(
	e common.EntityId,
	damage int64,
	hitpoints map[common.EntityId]*hitpoints,
) (dead bool, err error) {
	hpComp, ok := hitpoints[e]
	if !ok {
		return false, fmt.Errorf("could not get hitpoints component of entity %d", e)
	}

	hpComp.current -= damage

	if hpComp.current <= 0 {
		return true, nil
	}

	return false, nil
}

func (HitpointsManager) Heal(
	e common.EntityId,
	heal int64,
	hitpoints map[common.EntityId]*hitpoints,
) error {
	hpComp, ok := hitpoints[e]
	if !ok {
		return fmt.Errorf("could not get hitpoints component of entity %d", e)
	}

	hpComp.current += heal

	if hpComp.current > hpComp.max {
		hpComp.current = hpComp.max
	}

	return nil
}
