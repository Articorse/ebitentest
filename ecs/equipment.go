package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
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

type equipment struct {
	slot      EquipableSlotEnum
	abilities [data.MaxEquipmentAbilitySlots]EntityAbility
}

func (x equipment) Copy() equipment {
	copyAbilityParams(x.abilities[:])
	return x
}

func (x equipment) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Equipments.addComponent(e, x.Copy())
}

type equipmentDto struct {
	Slot      EquipableSlotEnum
	Abilities [data.MaxEquipmentAbilitySlots]EntityAbility
}

func (x equipment) toDto() (ComponentDto, error) {
	copyAbilityParams(x.abilities[:])
	return &equipmentDto{
		Slot:      x.slot,
		Abilities: x.abilities,
	}, nil
}

func (x equipmentDto) toComponent() (Component, error) {
	copyAbilityParams(x.Abilities[:])
	return &equipment{
		slot:      x.Slot,
		abilities: x.Abilities,
	}, nil
}
