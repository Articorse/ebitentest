package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
)

// Where an entity aims, such as at the cursor for a player, so that it and what it holds can turn and act towards it
type Aim struct {
	target    utils.Vec2f
	hasTarget bool
}

func NewAimComponent() *Aim {
	return &Aim{}
}

func (x *Aim) Target() (utils.Vec2f, bool) {
	return x.target, x.hasTarget
}

func (x *Aim) SetTarget(target utils.Vec2f) {
	x.target, x.hasTarget = target, true
}

func (x *Aim) ClearTarget() {
	x.target, x.hasTarget = utils.Vec2f{}, false
}

// The entity's own aim, or that of the closest entity it's attached to that has one, so that held items aim where
// their holder does
func (x *ECSContainer) AimOf(e common.EntityId) (utils.Vec2f, bool) {
	for ; e != -1; e = x.Parent(e) {
		if aim, ok := x.Aims.Get(e); ok {
			return aim.Target()
		}
	}

	return utils.Vec2f{}, false
}

func (x Aim) Copy() Aim {
	return x
}

func (x Aim) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Aims.addComponent(e, x.Copy())
}

type aimDto struct {
	Target    utils.Vec2f
	HasTarget bool
}

func (x Aim) toDto() (ComponentDto, error) {
	return &aimDto{Target: x.target, HasTarget: x.hasTarget}, nil
}

func (x aimDto) toComponent() (Component, error) {
	return &Aim{target: x.Target, hasTarget: x.HasTarget}, nil
}
