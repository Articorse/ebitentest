package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type contactDamageManager struct{}

func NewContactDamageComponent(
	source common.EntityId,
	knockback float64,
	dieOnContact bool,
	singleTick bool,
	damageTiers ...int,
) *contactDamage {
	return &contactDamage{
		source:       source,
		knockback:    knockback,
		dieOnContact: dieOnContact,
		singleTick:   singleTick,
		damageTiers:  damageTiers,
	}
}

func (*contactDamageManager) GetSource(
	e common.EntityId,
	ecs *ECS,
) (common.EntityId, error) {
	cdComp, err := ecs.ContactDamages.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get contact damage component of entity %d: %v", e, err)
	}

	return cdComp.source, nil
}

func (*contactDamageManager) GetDamageTiers(
	e common.EntityId,
	ecs *ECS,
) ([]int, error) {
	cdComp, err := ecs.ContactDamages.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get contact damage component of entity %d: %v", e, err)
	}

	return cdComp.damageTiers, nil
}

func (*contactDamageManager) GetKnockback(
	e common.EntityId,
	ecs *ECS,
) (float64, error) {
	cdComp, err := ecs.ContactDamages.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get contact damage component of entity %d: %v", e, err)
	}

	return cdComp.knockback, nil
}

func (*contactDamageManager) GetDieOnContact(
	e common.EntityId,
	ecs *ECS,
) (bool, error) {
	cdComp, err := ecs.ContactDamages.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get contact damage component of entity %d: %v", e, err)
	}

	return cdComp.dieOnContact, nil
}

func (*contactDamageManager) GetSingleTick(
	e common.EntityId,
	ecs *ECS,
) (bool, error) {
	cdComp, err := ecs.ContactDamages.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get contact damage component of entity %d: %v", e, err)
	}

	return cdComp.singleTick, nil
}
