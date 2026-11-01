package ecs

import "ebittest/ecs/common"

type Tether struct {
	anchor common.EntityId
}

func NewTetherComponent(anchor common.EntityId) *Tether {
	return &Tether{
		anchor: anchor,
	}
}

func (x *Tether) Anchor() common.EntityId {
	return x.anchor
}

func (x *Tether) SetAnchor(anchor common.EntityId) {
	x.anchor = anchor
}

func (x Tether) Copy() Tether {
	return Tether{
		anchor: x.anchor,
	}
}

func (x Tether) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Tethers.addComponent(e, x.Copy())
}

type tetherDto struct {
	Anchor common.EntityId
}

func (x Tether) toDto() (ComponentDto, error) {
	return &tetherDto{
		Anchor: x.anchor,
	}, nil
}

func (x tetherDto) toComponent() (Component, error) {
	return &Tether{
		anchor: x.Anchor,
	}, nil
}
