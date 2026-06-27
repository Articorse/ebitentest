package ecs

import "ebittest/data"

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

func (equipment) isComponent() {}

func (x equipment) Copy() equipment {
	abilitiesCopy := [data.MaxEquipmentAbilitySlots]EntityAbility{}

	for i, abi := range x.abilities {
		abilitiesCopy[i] = EntityAbility{
			Def:    abi.Def,
			Status: abi.Status,
		}
	}

	return equipment{
		slot:      x.slot,
		abilities: abilitiesCopy,
	}
}

type equipmentDto struct {
	Slot      EquipableSlotEnum
	Abilities [data.MaxEquipmentAbilitySlots]EntityAbility
}

func (equipmentDto) isComponentDto() {}

func (x equipment) ToDto() equipmentDto {
	return equipmentDto{
		Slot:      x.slot,
		Abilities: x.abilities,
	}
}

func (x equipmentDto) ToComponent() *equipment {
	return &equipment{
		slot:      x.Slot,
		abilities: x.Abilities,
	}
}
