package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
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
	sprites map[common.EntityId]*sprite,
) (dead bool, err error) {
	sm := SpriteManager{}

	hpComp, ok := hitpoints[e]
	if !ok {
		return false, fmt.Errorf("could not get hitpoints component of entity %d", e)
	}

	hpComp.current -= damage
	hpComp.invulCurMs = hpComp.invulMaxMs

	if hpComp.current <= 0 {
		return true, nil
	}

	sm.SetSpriteFlash(
		e,
		sprites,
		[]utils.RelativeColor{
			{R: 0.5, G: 0.5, B: 0.5, A: 1},
			{R: 1, G: 1, B: 1, A: 1},
		},
		[]uint64{100, 100},
		uint64(hpComp.invulMaxMs),
	)

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

	if hpComp.current > int64(hpComp.max) {
		hpComp.current = int64(hpComp.max)
	}

	return nil
}
