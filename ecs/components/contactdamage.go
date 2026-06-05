package components

import "ebittest/ecs/ecscommon"

type ContactDamage struct {
	source      ecscommon.EntityId
	damageTiers []int64
	knockback   float64
}

func (*ContactDamage) isComponent() {}

func (x ContactDamage) Copy() ContactDamage {
	return ContactDamage{
		source:      x.source,
		damageTiers: x.damageTiers,
		knockback:   x.knockback,
	}
}
