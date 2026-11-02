package ecs

import (
	"ebittest/ecs/common"
	"fmt"
	"slices"
)

// Every item in an inventory is also held by its owner, so that it's saved, streamed and dropped along with it
type Inventory struct {
	// -1 for an empty slot
	slots    []common.EntityId
	selected int
}

func NewInventoryComponent(capacity int) *Inventory {
	slots := make([]common.EntityId, max(capacity, 0))
	for i := range slots {
		slots[i] = -1
	}
	return &Inventory{slots: slots}
}

func (x *Inventory) Capacity() int {
	return len(x.slots)
}

func (x *Inventory) Slot(i int) (common.EntityId, bool) {
	if i < 0 || i >= len(x.slots) || x.slots[i] == -1 {
		return -1, false
	}
	return x.slots[i], true
}

func (x *Inventory) Selected() int {
	return x.selected
}

func (x *Inventory) Select(i int) error {
	if i < 0 || i >= len(x.slots) {
		return fmt.Errorf("slot %d is outside an inventory of %d slots", i, len(x.slots))
	}
	x.selected = i
	return nil
}

func (x *Inventory) slotOf(item common.EntityId) int {
	return slices.Index(x.slots, item)
}

// Holds the item in the given empty slot, attaching it to the owner if it isn't held by it already
func (x *ECSContainer) PutInInventory(owner common.EntityId, slot int, item common.EntityId) error {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return fmt.Errorf("entity %d has no inventory", owner)
	}
	if slot < 0 || slot >= len(inv.slots) {
		return fmt.Errorf("slot %d is outside the %d slots of entity %d's inventory", slot, len(inv.slots), owner)
	}
	if inv.slots[slot] != -1 {
		return fmt.Errorf("slot %d of entity %d's inventory already holds entity %d", slot, owner, inv.slots[slot])
	}
	if !x.Items.HasComponent(item) {
		return fmt.Errorf("entity %d isn't an item", item)
	}
	if i := inv.slotOf(item); i != -1 {
		return fmt.Errorf("entity %d is already in slot %d of entity %d's inventory", item, i, owner)
	}

	if x.Parent(item) != owner || x.LinkKind(item) != Link_Held {
		if err := x.Attach(item, owner, Link_Held); err != nil {
			return fmt.Errorf("error attaching item %d to entity %d: %w", item, owner, err)
		}
	}

	inv.slots[slot] = item
	return nil
}

// An item that stops being held by its owner leaves the owner's inventory with it
func (x *ECSContainer) takeOutOfInventory(owner, item common.EntityId) {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return
	}
	if i := inv.slotOf(item); i != -1 {
		inv.slots[i] = -1
	}
}

func (x Inventory) Copy() Inventory {
	return Inventory{slots: slices.Clone(x.slots), selected: x.selected}
}

func (x Inventory) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Inventories.addComponent(e, x.Copy())
}

type inventoryDto struct {
	Slots    []common.EntityId
	Selected int
}

func (x Inventory) toDto() (ComponentDto, error) {
	return &inventoryDto{Slots: slices.Clone(x.slots), Selected: x.selected}, nil
}

func (x inventoryDto) toComponent() (Component, error) {
	return &Inventory{slots: slices.Clone(x.Slots), selected: x.Selected}, nil
}
