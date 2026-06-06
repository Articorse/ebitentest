package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type ContactDamageManager struct{}

func NewContactDamageComponent(
	source common.EntityId,
	damageTiers []int64,
	knockback float64,
) *contactDamage {
	return &contactDamage{
		source:      source,
		damageTiers: damageTiers,
		knockback:   knockback,
	}
}

func (*ContactDamageManager) GetSource(
	e common.EntityId,
	contactDamages map[common.EntityId]*contactDamage,
) (common.EntityId, error) {
	cdComp, ok := contactDamages[e]
	if !ok {
		return -1, fmt.Errorf("could not get contact damage component of entity %d", e)
	}

	return cdComp.source, nil
}

func (*ContactDamageManager) GetDamageTiers(
	e common.EntityId,
	contactDamages map[common.EntityId]*contactDamage,
) ([]int64, error) {
	cdComp, ok := contactDamages[e]
	if !ok {
		return nil, fmt.Errorf("could not get contact damage component of entity %d", e)
	}

	return cdComp.damageTiers, nil
}

func (*ContactDamageManager) GetKnockback(
	e common.EntityId,
	contactDamages map[common.EntityId]*contactDamage,
) (float64, error) {
	cdComp, ok := contactDamages[e]
	if !ok {
		return -1, fmt.Errorf("could not get contact damage component of entity %d", e)
	}

	return cdComp.knockback, nil
}
