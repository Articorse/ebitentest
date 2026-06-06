package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type ContactDamageManager struct{}

func NewContactDamageComponent(
	source common.EntityId,
	knockback float64,
	dieOnContact bool,
	damageTiers ...int64,
) *contactDamage {
	return &contactDamage{
		source:       source,
		knockback:    knockback,
		dieOnContact: dieOnContact,
		damageTiers:  damageTiers,
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

func (*ContactDamageManager) GetDieOnContact(
	e common.EntityId,
	contactDamages map[common.EntityId]*contactDamage,
) (bool, error) {
	cdComp, ok := contactDamages[e]
	if !ok {
		return false, fmt.Errorf("could not get contact damage component of entity %d", e)
	}

	return cdComp.dieOnContact, nil
}
