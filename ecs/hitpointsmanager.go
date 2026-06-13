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
	world *World,
) (int64, error) {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	return hpComp.max, nil
}

// Returns -1 if component not found
func (HitpointsManager) GetCurrent(
	e common.EntityId,
	world *World,
) (int64, error) {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	return hpComp.current, nil
}

func (HitpointsManager) GetInvulMax(
	e common.EntityId,
	world *World,
) (int64, error) {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	return hpComp.invulMaxMs, nil
}

func (HitpointsManager) GetInvulCurrent(
	e common.EntityId,
	world *World,
) (int64, error) {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	return hpComp.invulCurMs, nil
}

func (HitpointsManager) SetInvul(
	e common.EntityId,
	time int64,
	world *World,
) error {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	hpComp.invulCurMs = time
	return nil
}

func (HitpointsManager) IsInvul(
	e common.EntityId,
	world *World,
) (bool, error) {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	return hpComp.invulCurMs > 0, nil
}

func (HitpointsManager) TickInvul(
	e common.EntityId,
	world *World,
) error {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
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
	world *World,
) (dead bool, err error) {
	sm := SpriteManager{}

	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	hpComp.current -= damage
	hpComp.invulCurMs = hpComp.invulMaxMs

	if hpComp.current <= 0 {
		return true, nil
	}

	err = sm.SetSpriteFlash(
		e,
		[]utils.RelativeColor{
			{R: 0.3, G: 0.3, B: 0.3, A: 1},
			{R: 1, G: 1, B: 1, A: 1},
		},
		[]uint64{100, 100},
		uint64(hpComp.invulMaxMs),
		world,
	)
	if err != nil {
		return false, fmt.Errorf("could not set sprite flash of entity %d: %v", e, err)
	}

	return false, nil
}

func (HitpointsManager) Heal(
	e common.EntityId,
	heal int64,
	world *World,
) error {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	hpComp.current += heal

	if hpComp.current > int64(hpComp.max) {
		hpComp.current = int64(hpComp.max)
	}

	return nil
}
