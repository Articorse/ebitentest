package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
	"maps"
	"slices"
)

type EquipSlotEnum uint8

const (
	Equip_MainHand EquipSlotEnum = iota
	Equip_OffHand
	Equip_Torso
	Equip_Arms
	Equip_Legs
	Equip_Ring1
	Equip_Ring2
	EquipSlotCount
)

var equipSlotNames = [EquipSlotCount]string{"Main hand", "Off hand", "Torso", "Arms", "Legs", "Ring 1", "Ring 2"}

func (x EquipSlotEnum) String() string {
	if x >= EquipSlotCount {
		return fmt.Sprintf("slot %d", x)
	}
	return equipSlotNames[x]
}

type Equipper struct {
	equipment map[EquipSlotEnum]common.EntityId
}

func NewEquipperComponent(equipment map[EquipSlotEnum]common.EntityId) *Equipper {
	return &Equipper{
		equipment: equipment,
	}
}

func (x *Equipper) Equip(slot EquipSlotEnum, e common.EntityId) {
	x.equipment[slot] = e
}

func (x *Equipper) Unequip(slot EquipSlotEnum) {
	delete(x.equipment, slot)
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

// The slot the item is equipped in, if it's equipped
func (x *Equipper) SlotOf(e common.EntityId) (EquipSlotEnum, bool) {
	for _, slot := range slices.Sorted(maps.Keys(x.equipment)) {
		if x.equipment[slot] == e {
			return slot, true
		}
	}
	return 0, false
}

// Whether the item is worn in a slot other than the main hand, which only follows what's selected on the hotbar
func (x *ECSContainer) IsWorn(owner, item common.EntityId) bool {
	equipper, ok := x.Equippers.Get(owner)
	if !ok {
		return false
	}
	slot, ok := equipper.SlotOf(item)
	return ok && slot != Equip_MainHand
}

// Whether the item can be worn by the owner in the slot, going by the slots it fits and the owner's kind of body
func (x *ECSContainer) CanWear(owner, item common.EntityId, slot EquipSlotEnum) bool {
	eq, ok := x.Equipments.Get(item)
	if !ok || slot == Equip_MainHand || slot >= EquipSlotCount || !eq.Fits(slot) {
		return false
	}
	kind := data.Body_None
	if body, ok := x.Bodies.Get(owner); ok {
		kind = body.Kind()
	}
	return eq.FitsBody(kind)
}

// Wears an item from the owner's inventory in the slot, taking it off the hotbar and out of any other slot, and taking
// off whatever was worn there, which stays in the inventory. Whether it could be worn there.
func (x *ECSContainer) Wear(owner, item common.EntityId, slot EquipSlotEnum) (bool, error) {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return false, fmt.Errorf("entity %d has no inventory", owner)
	}
	equipper, ok := x.Equippers.Get(owner)
	if !ok {
		return false, fmt.Errorf("entity %d has no equipper", owner)
	}
	if inv.slotOf(item) == -1 {
		return false, fmt.Errorf("entity %d isn't in entity %d's inventory", item, owner)
	}
	if !x.CanWear(owner, item, slot) {
		return false, nil
	}

	if h := inv.HotbarSlotOf(item); h != -1 {
		if err := x.BindHotbar(owner, h, -1); err != nil {
			return false, err
		}
	}
	if from, ok := equipper.SlotOf(item); ok {
		equipper.Unequip(from)
	}
	equipper.Equip(slot, item)
	return true, nil
}

// Takes off what's worn in the slot, which stays in the inventory
func (x *ECSContainer) TakeOff(owner common.EntityId, slot EquipSlotEnum) error {
	if slot == Equip_MainHand {
		return fmt.Errorf("the main hand of entity %d holds what's selected on its hotbar, so it can't be taken off", owner)
	}
	equipper, ok := x.Equippers.Get(owner)
	if !ok {
		return fmt.Errorf("entity %d has no equipper", owner)
	}
	equipper.Unequip(slot)
	return nil
}

// Takes the item off if it's worn, or else wears it in the first slot it can be worn in
func (x *ECSContainer) ToggleWorn(owner, item common.EntityId) error {
	equipper, ok := x.Equippers.Get(owner)
	if !ok {
		return fmt.Errorf("entity %d has no equipper", owner)
	}
	if slot, ok := equipper.SlotOf(item); ok && slot != Equip_MainHand {
		return x.TakeOff(owner, slot)
	}
	for slot := Equip_MainHand + 1; slot < EquipSlotCount; slot++ {
		if x.CanWear(owner, item, slot) {
			_, err := x.Wear(owner, item, slot)
			return err
		}
	}
	return nil
}

// What's drawn over the entity for what it wears, in slot order
func (x *ECSContainer) WornImages(e common.EntityId) []common.ImageAssetTag {
	equipper, ok := x.Equippers.Get(e)
	if !ok {
		return nil
	}
	images := []common.ImageAssetTag{}
	for _, slot := range slices.Sorted(maps.Keys(equipper.equipment)) {
		if eq, ok := x.Equipments.Get(equipper.equipment[slot]); ok && eq.worn != "" {
			images = append(images, eq.worn)
		}
	}
	return images
}
