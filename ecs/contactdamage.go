package ecs

import "ebittest/ecs/common"

type contactDamage struct {
	source       common.EntityId
	knockback    float64
	dieOnContact bool
	damageTiers  []int
}

func (contactDamage) isComponent() {}

func (x contactDamage) Copy() contactDamage {
	return contactDamage{
		source:       x.source,
		knockback:    x.knockback,
		dieOnContact: x.dieOnContact,
		damageTiers:  x.damageTiers,
	}
}
