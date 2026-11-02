package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
)

// Carries an entity along the ground in a shallow arc, which is only drawn: the entity stays on the ground the whole
// way, so it's sorted and collides where it lands
type Toss struct {
	// Ground movement each tick
	step       utils.Vec2f
	durationMs int
	elapsedMs  int
	heightPx   float64
}

func NewTossComponent(distance utils.Vec2f, durationMs int, heightPx float64) *Toss {
	// Rounded up, like the number of ticks the toss runs for
	ticks := max((durationMs+data.TickMs-1)/data.TickMs, 1)
	return &Toss{step: distance.Multiply(1 / float64(ticks)), durationMs: durationMs, heightPx: heightPx}
}

// Moves the entity one tick along its toss. Each axis is blocked on its own, so that meeting terrain stops the part of
// the movement towards it, while the rest of the toss, and its arc, carries on.
func (x *ECSContainer) TickToss(e common.EntityId) {
	toss, ok := x.Tosses.Get(e)
	if !ok {
		return
	}
	t, ok := x.Transforms.Get(e)
	if !ok {
		x.EndToss(e)
		return
	}

	pos := t.LocalPos()
	if next := pos.Add(utils.Vec2f{X: toss.step.X}); toss.step.X != 0 && x.terrain.IsBlockedAt(next) {
		toss.step.X = 0
	} else {
		pos = next
	}
	if next := pos.Add(utils.Vec2f{Y: toss.step.Y}); toss.step.Y != 0 && x.terrain.IsBlockedAt(next) {
		toss.step.Y = 0
	} else {
		pos = next
	}
	t.SetLocalPos(pos)

	toss.elapsedMs += data.TickMs
	if toss.elapsedMs >= toss.durationMs {
		x.EndToss(e)
		return
	}

	if spr, ok := x.Sprites.Get(e); ok {
		progress := float64(toss.elapsedMs) / float64(toss.durationMs)
		spr.SetLocalOffsetPos(utils.Vec2f{Y: -4 * toss.heightPx * progress * (1 - progress)})
	}
}

// Puts the entity down where it is
func (x *ECSContainer) EndToss(e common.EntityId) {
	if !x.Tosses.HasComponent(e) {
		return
	}
	x.Tosses.deleteEntity(e)
	if spr, ok := x.Sprites.Get(e); ok {
		spr.SetLocalOffsetPos(utils.Vec2f{})
	}
}

func (x Toss) Copy() Toss {
	return x
}

func (x Toss) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Tosses.addComponent(e, x.Copy())
}

type tossDto struct {
	Step       utils.Vec2f
	DurationMs int
	ElapsedMs  int
	HeightPx   float64
}

func (x Toss) toDto() (ComponentDto, error) {
	return &tossDto{Step: x.step, DurationMs: x.durationMs, ElapsedMs: x.elapsedMs, HeightPx: x.heightPx}, nil
}

func (x tossDto) toComponent() (Component, error) {
	return &Toss{step: x.Step, durationMs: x.DurationMs, elapsedMs: x.ElapsedMs, heightPx: x.HeightPx}, nil
}
