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
	damageTiers ...int,
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
	world *World,
) (common.EntityId, error) {
	cdComp, err := world.ContactDamages.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get contact damage component of entity %d: %v", e, err)
	}

	return cdComp.source, nil
}

func (*ContactDamageManager) GetDamageTiers(
	e common.EntityId,
	world *World,
) ([]int, error) {
	cdComp, err := world.ContactDamages.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get contact damage component of entity %d: %v", e, err)
	}

	return cdComp.damageTiers, nil
}

func (*ContactDamageManager) GetKnockback(
	e common.EntityId,
	world *World,
) (float64, error) {
	cdComp, err := world.ContactDamages.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get contact damage component of entity %d: %v", e, err)
	}

	return cdComp.knockback, nil
}

func (*ContactDamageManager) GetDieOnContact(
	e common.EntityId,
	world *World,
) (bool, error) {
	cdComp, err := world.ContactDamages.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get contact damage component of entity %d: %v", e, err)
	}

	return cdComp.dieOnContact, nil
}
