package ecs

import "ebittest/ecs/common"

type contactDamage struct {
	source      common.EntityId
	damageTiers []int64
	knockback   float64
}

func (*contactDamage) isComponent() {}

func (x contactDamage) Copy() contactDamage {
	return contactDamage{
		source:      x.source,
		damageTiers: x.damageTiers,
		knockback:   x.knockback,
	}
}
