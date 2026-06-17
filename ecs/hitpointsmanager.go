package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type hitpointsManager struct{}

func NewHitpointsComponent(max int, invul int) *hitpoints {
	return &hitpoints{
		max:            max,
		current:        max,
		postHitInvulMs: invul,
		invulCurMs:     0,
	}
}

func (hitpointsManager) GetMax(
	e common.EntityId,
	world *World,
) (int, error) {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	return hpComp.max, nil
}

// Returns -1 if component not found
func (hitpointsManager) GetCurrent(
	e common.EntityId,
	world *World,
) (int, error) {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	return hpComp.current, nil
}

func (hitpointsManager) GetInvulMax(
	e common.EntityId,
	world *World,
) (int, error) {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	return hpComp.postHitInvulMs, nil
}

func (hitpointsManager) GetInvulCurrent(
	e common.EntityId,
	world *World,
) (int, error) {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	return hpComp.invulCurMs, nil
}

func (hitpointsManager) SetInvul(
	e common.EntityId,
	time int,
	world *World,
) error {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	hpComp.invulCurMs = time
	return nil
}

func (hitpointsManager) IsInvul(
	e common.EntityId,
	world *World,
) (bool, error) {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	return hpComp.invulCurMs > 0, nil
}

func (hitpointsManager) TickInvul(
	e common.EntityId,
	world *World,
) error {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	if hpComp.invulCurMs > 0 {
		hpComp.invulCurMs -= data.TickMs
	}

	if hpComp.invulCurMs < 0 {
		hpComp.invulCurMs = 0
	}

	return nil
}

// TODO: Add immobility time similar to invulnerability time
func (hitpointsManager) TakeDamage(
	e common.EntityId,
	damage int,
	world *World,
) (dead bool, err error) {
	sm := spriteManager{}

	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	hpComp.current -= damage

	if hpComp.invulCurMs < hpComp.postHitInvulMs {
		hpComp.invulCurMs = hpComp.postHitInvulMs
	}

	if hpComp.current <= 0 {
		return true, nil
	}

	err = sm.SetSpriteFlash(
		e,
		[]utils.RelativeColor{
			{R: 0.3, G: 0.3, B: 0.3, A: 1},
			{R: 1, G: 1, B: 1, A: 1},
		},
		[]int{100, 100},
		hpComp.postHitInvulMs,
		world,
	)
	if err != nil {
		return false, fmt.Errorf("could not set sprite flash of entity %d: %v", e, err)
	}

	return false, nil
}

func (hitpointsManager) Heal(
	e common.EntityId,
	heal int,
	world *World,
) error {
	hpComp, err := world.Hitpoints.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get hitpoints component of entity %d: %v", e, err)
	}

	hpComp.current += heal

	if hpComp.current > hpComp.max {
		hpComp.current = hpComp.max
	}

	return nil
}
