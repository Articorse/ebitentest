package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
)

type HitpointsManager struct{}

func NewHitpointsComponent(max int64, invul int64) *hitpoints {
	return &hitpoints{
		max:        max,
		current:    max,
		invulMaxMs: invul,
		invulCurMs: 0,
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

func (HitpointsManager) GetInvulMax(
	e common.EntityId,
	hitpoints map[common.EntityId]*hitpoints,
) (int64, error) {
	hpComp, ok := hitpoints[e]
	if !ok {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d", e)
	}

	return hpComp.invulMaxMs, nil
}

func (HitpointsManager) GetInvulCurrent(
	e common.EntityId,
	hitpoints map[common.EntityId]*hitpoints,
) (int64, error) {
	hpComp, ok := hitpoints[e]
	if !ok {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d", e)
	}

	return hpComp.invulCurMs, nil
}

func (HitpointsManager) IsInvul(
	e common.EntityId,
	hitpoints map[common.EntityId]*hitpoints,
) (bool, error) {
	hpComp, ok := hitpoints[e]
	if !ok {
		return false, fmt.Errorf("could not get hitpoints component of entity %d", e)
	}

	return hpComp.invulCurMs > 0, nil
}

func (HitpointsManager) TickInvul(
	e common.EntityId,
	hitpoints map[common.EntityId]*hitpoints,
) error {
	hpComp, ok := hitpoints[e]
	if !ok {
		return fmt.Errorf("could not get hitpoints component of entity %d", e)
	}

	if hpComp.invulCurMs > 0 {
		hpComp.invulCurMs -= 1000 / data.TPS
	}

	if hpComp.invulCurMs < 0 {
		hpComp.invulCurMs = 0
	}

	return nil
}

// TODO: Add immobility time similar to invulnerability time
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
	hpComp.invulCurMs = hpComp.invulMaxMs

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
