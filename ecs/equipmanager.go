package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type equipManager struct{}

func NewEquipmentComponent(slot EquipableSlotEnum, abilities [data.MaxEquipmentAbilitySlots]EntityAbility) *equipment {
	abis := [data.MaxEquipmentAbilitySlots]EntityAbility{}

	for i, abi := range abilities {
		abis[i] = EntityAbility{
			Name:   abi.Name,
			Def:    abi.Def,
			Status: AbilityStatus{State: AbiAct_Ready},
		}
	}

	return &equipment{
		slot:      slot,
		abilities: abis,
	}
}

func NewEquipperComponent(equipment map[EquipSlotEnum]common.EntityId) *equipper {
	return &equipper{
		equipment: equipment,
	}
}

func (equipManager) GetEquipmentEntities(
	e common.EntityId,
	ecs *ECS,
) ([]common.EntityId, error) {
	equipperComp, err := ecs.Equippers.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get equipper component of entity %d: %v", e, err)
	}

	equipmentEntities := make([]common.EntityId, 0, len(equipperComp.equipment))
	for _, eqE := range equipperComp.equipment {
		equipmentEntities = append(equipmentEntities, eqE)
	}

	return equipmentEntities, nil
}

func (equipManager) GetEquipmentInSlot(
	e common.EntityId,
	slot EquipSlotEnum,
	ecs *ECS,
) (eq common.EntityId, hasEqInSlot bool, err error) {
	equipperComp, err := ecs.Equippers.getComponent(e)
	if err != nil {
		return -1, false, fmt.Errorf("could not get equipper component of entity %d: %v", e, err)
	}

	equipId, ok := equipperComp.equipment[slot]
	if !ok {
		return -1, false, nil
	}

	return equipId, true, nil
}

func (equipManager) ActivateAbility(
	e common.EntityId,
	slot EquipSlotEnum,
	targets []common.EntityId,
	targetPos utils.Vec2,
	abiIdx int,
	ecs *ECS,
) (activated bool, err error) {
	if abiIdx > data.MaxEquipmentAbilitySlots-1 {
		return false, fmt.Errorf("ability index %d is out of bounds", abiIdx)
	}

	em := equipManager{}

	eqE, hasEq, err := em.GetEquipmentInSlot(e, slot, ecs)
	if err != nil {
		return false, fmt.Errorf("error getting equipment in slot %v of entity %d: %v", slot, e, err)
	}

	if !hasEq {
		return false, nil
	}

	equipmentComp, err := ecs.Equipments.getComponent(eqE)
	if err != nil {
		return false, fmt.Errorf("could not get equipment component of entity %d: %v", eqE, err)
	}

	abi := equipmentComp.abilities[abiIdx]

	_, err = tryActivate(eqE, &abi, targets, targetPos, ecs)
	if err != nil {
		return false, fmt.Errorf("error trying to activate ability %v of equipment entity %d: %v", abi.Name, eqE, err)
	}

	equipmentComp.abilities[abiIdx] = abi

	return true, nil
}

func (equipManager) TickAbilities(e common.EntityId, ecs *ECS) error {
	equipmentComp, err := ecs.Equipments.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get equipment component of entity %d: %v", e, err)
	}

	for i, a := range equipmentComp.abilities {
		err := tickAbilityState(e, &a, ecs)
		if err != nil {
			return fmt.Errorf("error ticking ability %v of equipment entity %d: %v", a.Name, e, err)
		}

		equipmentComp.abilities[i] = a
	}

	return nil
}
