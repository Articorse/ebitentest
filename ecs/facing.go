package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
)

// Turns the entity towards where it aims. Turns instantly and all the way around unless told otherwise.
type Facing struct {
	// Radians per second
	turnRate float64

	limited bool
	// Around the parent's rotation, or around 0 without a parent
	arcCenter float64
	arcHalf   float64
}

func NewFacingComponent() *Facing {
	return &Facing{}
}

func (x *Facing) WithTurnRate(radiansPerSecond float64) *Facing {
	x.turnRate = radiansPerSecond
	return x
}

// Only faces within half the given arc either side of the center, relative to the parent's rotation
func (x *Facing) WithAngleLimits(center, arc float64) *Facing {
	x.limited, x.arcCenter, x.arcHalf = true, center, arc/2
	return x
}

// The local rotation after one tick of turning from current towards target, both local rotations
func (x *Facing) Turn(current, target float64) float64 {
	maxStep := x.turnRate * data.TickMs / 1000

	if !x.limited {
		return utils.NormalizeAngle(current + x.step(utils.NormalizeAngle(target-current), maxStep))
	}

	// Measured from the arc's center and kept inside it, so that the turn never goes the short way round through the
	// part it can't face
	from := x.clampToArc(utils.NormalizeAngle(current - x.arcCenter))
	to := x.clampToArc(utils.NormalizeAngle(target - x.arcCenter))
	return utils.NormalizeAngle(x.arcCenter + from + x.step(to-from, maxStep))
}

func (x *Facing) step(by, maxStep float64) float64 {
	if x.turnRate <= 0 {
		return by
	}
	return min(max(by, -maxStep), maxStep)
}

func (x *Facing) clampToArc(a float64) float64 {
	return min(max(a, -x.arcHalf), x.arcHalf)
}

func (x Facing) Copy() Facing {
	return x
}

func (x Facing) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Facings.addComponent(e, x.Copy())
}

type facingDto struct {
	TurnRate  float64
	Limited   bool
	ArcCenter float64
	ArcHalf   float64
}

func (x Facing) toDto() (ComponentDto, error) {
	return &facingDto{TurnRate: x.turnRate, Limited: x.limited, ArcCenter: x.arcCenter, ArcHalf: x.arcHalf}, nil
}

func (x facingDto) toComponent() (Component, error) {
	return &Facing{turnRate: x.TurnRate, limited: x.Limited, arcCenter: x.ArcCenter, arcHalf: x.ArcHalf}, nil
}
