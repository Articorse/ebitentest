package components

import (
	"ebittest/ecs/ecscommon"
	"fmt"
)

type ContactDamageManager struct{}

func NewContactDamageComponent(
	source ecscommon.EntityId,
	damageTiers []int64,
	knockback float64,
) *ContactDamage {
	return &ContactDamage{
		source:      source,
		damageTiers: damageTiers,
		knockback:   knockback,
	}
}

func (*ContactDamageManager) GetSource(
	e ecscommon.EntityId,
	contactDamages map[ecscommon.EntityId]*ContactDamage,
) (ecscommon.EntityId, error) {
	cdComp, ok := contactDamages[e]
	if !ok {
		return -1, fmt.Errorf("could not get contact damage component of entity %d", e)
	}

	return cdComp.source, nil
}

func (*ContactDamageManager) GetDamageTiers(
	e ecscommon.EntityId,
	contactDamages map[ecscommon.EntityId]*ContactDamage,
) ([]int64, error) {
	cdComp, ok := contactDamages[e]
	if !ok {
		return nil, fmt.Errorf("could not get contact damage component of entity %d", e)
	}

	return cdComp.damageTiers, nil
}

func (*ContactDamageManager) GetKnockback(
	e ecscommon.EntityId,
	contactDamages map[ecscommon.EntityId]*ContactDamage,
) (float64, error) {
	cdComp, ok := contactDamages[e]
	if !ok {
		return -1, fmt.Errorf("could not get contact damage component of entity %d", e)
	}

	return cdComp.knockback, nil
}
