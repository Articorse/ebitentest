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
