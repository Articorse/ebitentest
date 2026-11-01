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

func (x equipper) Copy() equipper {
	equipmentCopy := make(map[EquipSlotEnum]common.EntityId)
	for slot, entityId := range x.equipment {
		equipmentCopy[slot] = entityId
	}

	return equipper{
		equipment: equipmentCopy,
	}
}

func (x equipper) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Equippers.addComponent(e, x.Copy())
}

type equipperDto struct {
	Equipment map[EquipSlotEnum]common.EntityId
}

func (x equipper) toDto() (ComponentDto, error) {
	x = x.Copy()

	return &equipperDto{
		Equipment: x.equipment,
	}, nil
}

func (x *equipperDto) toComponent() (Component, error) {
	return &equipper{
		equipment: x.Equipment,
	}, nil
}
