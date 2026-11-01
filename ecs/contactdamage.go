package ecs

import (
	"ebittest/ecs/common"
	"slices"
)

type ContactDamage struct {
	source       common.EntityId
	knockback    float64
	dieOnContact bool
	singleTick   bool
	damageTiers  []int
}

func NewContactDamageComponent(
	source common.EntityId,
	knockback float64,
	dieOnContact bool,
	singleTick bool,
	damageTiers ...int,
) *ContactDamage {
	return &ContactDamage{
		source:       source,
		knockback:    knockback,
		dieOnContact: dieOnContact,
		singleTick:   singleTick,
		damageTiers:  damageTiers,
	}
}

func (x *ContactDamage) Source() common.EntityId {
	return x.source
}

// A copy, so that callers can't change the tiers of the component
func (x *ContactDamage) DamageTiers() []int {
	return slices.Clone(x.damageTiers)
}

func (x *ContactDamage) Knockback() float64 {
	return x.knockback
}

func (x *ContactDamage) DieOnContact() bool {
	return x.dieOnContact
}

func (x *ContactDamage) SingleTick() bool {
	return x.singleTick
}

func (x ContactDamage) Copy() ContactDamage {
	return ContactDamage{
		source:       x.source,
		knockback:    x.knockback,
		dieOnContact: x.dieOnContact,
		singleTick:   x.singleTick,
		damageTiers:  slices.Clone(x.damageTiers),
	}
}

func (x ContactDamage) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.ContactDamages.addComponent(e, x.Copy())
}

type contactDamageDto struct {
	Source       common.EntityId
	Knockback    float64
	DieOnContact bool
	SingleTick   bool
	DamageTiers  []int
}

func (x ContactDamage) toDto() (ComponentDto, error) {
	x = x.Copy()
	return &contactDamageDto{
		Source:       x.source,
		Knockback:    x.knockback,
		DieOnContact: x.dieOnContact,
		SingleTick:   x.singleTick,
		DamageTiers:  x.damageTiers,
	}, nil
}

func (x contactDamageDto) toComponent() (Component, error) {
	return &ContactDamage{
		source:       x.Source,
		knockback:    x.Knockback,
		dieOnContact: x.DieOnContact,
		singleTick:   x.SingleTick,
		damageTiers:  x.DamageTiers,
	}, nil
}
