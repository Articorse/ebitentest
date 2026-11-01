package ecs

import (
	"ebittest/ecs/common"
	"maps"
	"slices"
)

type EquipSlotEnum uint8

const (
	Equip_MainHand EquipSlotEnum = iota
	Equip_OffHand
	Equip_Body
	Equip_Arms
	Equip_Ring1
	Equip_Ring2
)

type Equipper struct {
	equipment map[EquipSlotEnum]common.EntityId
}

func NewEquipperComponent(equipment map[EquipSlotEnum]common.EntityId) *Equipper {
	return &Equipper{
		equipment: equipment,
	}
}

// In slot order, so that iterating them doesn't depend on map order
func (x *Equipper) Equipment() []common.EntityId {
	equipment := make([]common.EntityId, 0, len(x.equipment))
	for _, slot := range slices.Sorted(maps.Keys(x.equipment)) {
		equipment = append(equipment, x.equipment[slot])
	}

	return equipment
}

func (x *Equipper) InSlot(slot EquipSlotEnum) (common.EntityId, bool) {
	e, ok := x.equipment[slot]
	return e, ok
}

func (x Equipper) Copy() Equipper {
	equipmentCopy := make(map[EquipSlotEnum]common.EntityId)
	for slot, entityId := range x.equipment {
		equipmentCopy[slot] = entityId
	}

	return Equipper{
		equipment: equipmentCopy,
	}
}

func (x Equipper) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Equippers.addComponent(e, x.Copy())
}

type equipperDto struct {
	Equipment map[EquipSlotEnum]common.EntityId
}

func (x Equipper) toDto() (ComponentDto, error) {
	x = x.Copy()

	return &equipperDto{
		Equipment: x.equipment,
	}, nil
}

func (x *equipperDto) toComponent() (Component, error) {
	return &Equipper{
		equipment: x.Equipment,
	}, nil
}
