package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type EquipableSlotEnum uint8

const (
	Equipable_MainHand = EquipableSlotEnum(0b00000001)
	Equipable_OffHand  = EquipableSlotEnum(0b00000010)
	Equipable_Body     = EquipableSlotEnum(0b00000100)
	Equipable_Arms     = EquipableSlotEnum(0b00001000)
	Equipable_Ring1    = EquipableSlotEnum(0b00010000)
	Equipable_Ring2    = EquipableSlotEnum(0b00100000)
)

type Equipment struct {
	slot      EquipableSlotEnum
	abilities [data.MaxEquipmentAbilitySlots]EntityAbility
}

func NewEquipmentComponent(slot EquipableSlotEnum, abilities [data.MaxEquipmentAbilitySlots]EntityAbility) *Equipment {
	abis := [data.MaxEquipmentAbilitySlots]EntityAbility{}

	for i, abi := range abilities {
		abis[i] = EntityAbility{
			Def:    abi.Def,
			Status: AbilityStatus{State: AbiAct_Ready},
			Params: abi.Params,
		}
	}

	return &Equipment{
		slot:      slot,
		abilities: abis,
	}
}

func (x *ECSContainer) ActivateEquipmentAbility(
	e common.EntityId,
	slot EquipSlotEnum,
	targets []common.EntityId,
	targetPos utils.Vec2f,
	abiIdx int,
) (activated bool, err error) {
	if abiIdx > data.MaxEquipmentAbilitySlots-1 {
		return false, fmt.Errorf("ability index %d is out of bounds", abiIdx)
	}

	equipper, ok := x.Equippers.Get(e)
	if !ok {
		return false, fmt.Errorf("entity %d has no equipper", e)
	}

	eqE, ok := equipper.InSlot(slot)
	if !ok {
		return false, nil
	}

	eq, ok := x.Equipments.Get(eqE)
	if !ok {
		return false, fmt.Errorf("entity %d in slot %v of entity %d has no equipment", eqE, slot, e)
	}

	// Activated on a copy, so that a failed activation leaves the ability as it was
	abi := eq.abilities[abiIdx]
	if _, err := tryActivate(eqE, &abi, targets, targetPos, x); err != nil {
		return false, fmt.Errorf("error trying to activate ability %v of equipment entity %d: %v", abi.Def.AbilityId, eqE, err)
	}
	eq.abilities[abiIdx] = abi

	return true, nil
}

func (x *ECSContainer) TickEquipmentAbilities(e common.EntityId) error {
	eq, ok := x.Equipments.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no equipment", e)
	}

	for i, a := range eq.abilities {
		if err := tickAbilityState(e, &a, x); err != nil {
			return fmt.Errorf("error ticking ability %v of equipment entity %d: %v", a.Def.AbilityId, e, err)
		}
		eq.abilities[i] = a
	}

	return nil
}

func (x Equipment) Copy() Equipment {
	copyAbilityParams(x.abilities[:])
	return x
}

func (x Equipment) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Equipments.addComponent(e, x.Copy())
}

type equipmentDto struct {
	Slot      EquipableSlotEnum
	Abilities [data.MaxEquipmentAbilitySlots]EntityAbility
}

func (x Equipment) toDto() (ComponentDto, error) {
	copyAbilityParams(x.abilities[:])
	return &equipmentDto{
		Slot:      x.slot,
		Abilities: x.abilities,
	}, nil
}

func (x equipmentDto) toComponent() (Component, error) {
	copyAbilityParams(x.Abilities[:])
	return &Equipment{
		slot:      x.Slot,
		abilities: x.Abilities,
	}, nil
}
