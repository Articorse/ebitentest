package ecs

import "ebittest/ecs/common"

type EquipSlotEnum uint8

const (
	Equip_MainHand EquipSlotEnum = iota
	Equip_OffHand
	Equip_Body
	Equip_Arms
	Equip_Ring1
	Equip_Ring2
)

type equipper struct {
	equipment map[EquipSlotEnum]common.EntityId
}

func (equipper) isComponent() {}

func (x equipper) Copy() equipper {
	equipmentCopy := make(map[EquipSlotEnum]common.EntityId)
	for slot, entityId := range x.equipment {
		equipmentCopy[slot] = entityId
	}

	return equipper{
		equipment: equipmentCopy,
	}
}

type equipperDto struct {
	Equipment map[EquipSlotEnum]common.EntityId
}

func (equipperDto) isComponentDto() {}

func (x equipper) ToDto() equipperDto {
	return equipperDto{
		Equipment: x.equipment,
	}
}

func (x *equipperDto) ToComponent() *equipper {
	return &equipper{
		equipment: x.Equipment,
	}
}
