package ecs

import "ebittest/ecs/common"

type contactDamage struct {
	source       common.EntityId
	knockback    float64
	dieOnContact bool
	singleTick   bool
	damageTiers  []int
}

func (x contactDamage) Copy() contactDamage {
	dTiersCopy := make([]int, len(x.damageTiers))
	copy(dTiersCopy, x.damageTiers)

	return contactDamage{
		source:       x.source,
		knockback:    x.knockback,
		dieOnContact: x.dieOnContact,
		singleTick:   x.singleTick,
		damageTiers:  dTiersCopy,
	}
}

func (x contactDamage) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.ContactDamages.addComponent(e, x.Copy())
}

type contactDamageDto struct {
	Source       common.EntityId
	Knockback    float64
	DieOnContact bool
	SingleTick   bool
	DamageTiers  []int
}

func (x contactDamage) toDto() (ComponentDto, error) {
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
	return &contactDamage{
		source:       x.Source,
		knockback:    x.Knockback,
		dieOnContact: x.DieOnContact,
		singleTick:   x.SingleTick,
		damageTiers:  x.DamageTiers,
	}, nil
}
