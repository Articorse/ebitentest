package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type Hitpoints struct {
	max            int
	current        int
	postHitInvulMs int
	invulCurMs     int
}

func NewHitpointsComponent(max int, invul int) *Hitpoints {
	return &Hitpoints{
		max:            max,
		current:        max,
		postHitInvulMs: invul,
		invulCurMs:     0,
	}
}

func (x *Hitpoints) Max() int {
	return x.max
}

func (x *Hitpoints) Current() int {
	return x.current
}

func (x *Hitpoints) InvulMax() int {
	return x.postHitInvulMs
}

func (x *Hitpoints) InvulCurrent() int {
	return x.invulCurMs
}

func (x *Hitpoints) SetInvul(ms int) {
	x.invulCurMs = ms
}

func (x *Hitpoints) IsInvul() bool {
	return x.invulCurMs > 0
}

func (x *Hitpoints) TickInvul() {
	if x.invulCurMs > 0 {
		x.invulCurMs -= data.TickMs
	}

	if x.invulCurMs < 0 {
		x.invulCurMs = 0
	}
}

// TODO: Add immobility time similar to invulnerability time
func (x *Hitpoints) TakeDamage(damage int) (dead bool) {
	x.current -= damage

	if x.invulCurMs < x.postHitInvulMs {
		x.invulCurMs = x.postHitInvulMs
	}

	return x.current <= 0
}

func (x *Hitpoints) Heal(heal int) {
	x.current += heal

	if x.current > x.max {
		x.current = x.max
	}
}

// Damages e and flashes its sprite, if it has one, for as long as the damage makes it invulnerable
func (x *ECSContainer) Damage(e common.EntityId, damage int) (dead bool, err error) {
	hp, ok := x.Hitpoints.Get(e)
	if !ok {
		return false, fmt.Errorf("entity %d has no hitpoints", e)
	}

	if hp.TakeDamage(damage) {
		return true, nil
	}

	if s, ok := x.Sprites.Get(e); ok {
		err := s.SetFlash(
			[]utils.RelativeColor{
				{R: 0.3, G: 0.3, B: 0.3, A: 1},
				{R: 1, G: 1, B: 1, A: 1},
			},
			[]int{100, 100},
			hp.postHitInvulMs,
		)
		if err != nil {
			return false, fmt.Errorf("could not set sprite flash of entity %d: %v", e, err)
		}
	}

	return false, nil
}

func (x Hitpoints) Copy() Hitpoints {
	return Hitpoints{
		max:            x.max,
		current:        x.current,
		postHitInvulMs: x.postHitInvulMs,
		invulCurMs:     x.invulCurMs,
	}
}

func (x Hitpoints) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Hitpoints.addComponent(e, x.Copy())
}

type hitpointsDto struct {
	Max            int
	Current        int
	PostHitInvulMs int
	InvulCurMs     int
}

func (x Hitpoints) toDto() (ComponentDto, error) {
	return &hitpointsDto{
		Max:            x.max,
		Current:        x.current,
		PostHitInvulMs: x.postHitInvulMs,
		InvulCurMs:     x.invulCurMs,
	}, nil
}

func (x hitpointsDto) toComponent() (Component, error) {
	return &Hitpoints{
		max:            x.Max,
		current:        x.Current,
		postHitInvulMs: x.PostHitInvulMs,
		invulCurMs:     x.InvulCurMs,
	}, nil
}
