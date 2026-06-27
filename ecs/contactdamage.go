package ecs

import "ebittest/ecs/common"

type contactDamage struct {
	source       common.EntityId
	knockback    float64
	dieOnContact bool
	singleTick   bool
	damageTiers  []int
}

func (contactDamage) isComponent() {}

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

type contactDamageDto struct {
	Source       common.EntityId
	Knockback    float64
	DieOnContact bool
	SingleTick   bool
	DamageTiers  []int
}

func (contactDamageDto) isComponentDto() {}

func (x contactDamage) ToDto() contactDamageDto {
	return contactDamageDto{
		Source:       x.source,
		Knockback:    x.knockback,
		DieOnContact: x.dieOnContact,
		SingleTick:   x.singleTick,
		DamageTiers:  x.damageTiers,
	}
}

func (x contactDamageDto) ToComponent() *contactDamage {
	return &contactDamage{
		source:       x.Source,
		knockback:    x.Knockback,
		dieOnContact: x.DieOnContact,
		singleTick:   x.SingleTick,
		damageTiers:  x.DamageTiers,
	}
}
