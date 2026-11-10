package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"slices"
)

// Every item in an inventory is also held by its owner, so that it's saved, streamed and dropped along with it. The
// hotbar isn't more room: each of its slots points at an item in the inventory, wherever that item is moved to.
type Inventory struct {
	// -1 for an empty slot
	slots []common.EntityId
	// -1 for a hotbar slot that points at nothing
	hotbar []common.EntityId
	// The hotbar slot whose item is in hand
	selected int
	// Leaves what it picks up off its hotbar, which a colonist only uses for what it's using at the moment
	pickupsOffHotbar bool
	// In milligrams
	maxMass int
}

func NewInventoryComponent(capacity, hotbarSize, maxMassMg int) *Inventory {
	return &Inventory{slots: emptySlots(capacity), hotbar: emptySlots(hotbarSize), maxMass: maxMassMg}
}

func emptySlots(n int) []common.EntityId {
	slots := make([]common.EntityId, max(n, 0))
	for i := range slots {
		slots[i] = -1
	}
	return slots
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

// -1 if every slot holds an item
func (x *Inventory) FirstFreeSlot() int {
	return slices.Index(x.slots, -1)
}

func (x *Inventory) HotbarSize() int {
	return len(x.hotbar)
}

func (x *Inventory) HotbarItem(i int) (common.EntityId, bool) {
	if i < 0 || i >= len(x.hotbar) || x.hotbar[i] == -1 {
		return -1, false
	}
	return x.hotbar[i], true
}

// -1 if the item isn't on the hotbar
func (x *Inventory) HotbarSlotOf(item common.EntityId) int {
	return slices.Index(x.hotbar, item)
}

// The hotbar slot whose item is in hand
func (x *Inventory) Selected() int {
	return x.selected
}

func (x *Inventory) slotOf(item common.EntityId) int {
	return slices.Index(x.slots, item)
}

// Holds the item in the given empty slot, attaching it to the owner if it isn't held by it already. It isn't put on the
// hotbar.
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
	x.SetDisabled(item, true)
	return x.refreshHand(owner)
}

// Points the hotbar slot at an item in the inventory, or at nothing for -1. An item is only ever on one hotbar slot, so
// binding it again moves it. What's worn has to be taken off first, so that it can't be held as well.
func (x *ECSContainer) BindHotbar(owner common.EntityId, hotbarSlot int, item common.EntityId) error {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return fmt.Errorf("entity %d has no inventory", owner)
	}
	if hotbarSlot < 0 || hotbarSlot >= len(inv.hotbar) {
		return fmt.Errorf("hotbar slot %d is outside the %d of entity %d's inventory", hotbarSlot, len(inv.hotbar), owner)
	}
	if item != -1 {
		if inv.slotOf(item) == -1 {
			return fmt.Errorf("entity %d isn't in entity %d's inventory", item, owner)
		}
		if x.IsWorn(owner, item) {
			return nil
		}
		if i := inv.HotbarSlotOf(item); i != -1 {
			inv.hotbar[i] = -1
		}
	}

	inv.hotbar[hotbarSlot] = item
	return x.refreshHand(owner)
}

// Binds the item to the first hotbar slot that points at nothing, if there is one
func (x *ECSContainer) bindToFreeHotbarSlot(owner, item common.EntityId) error {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return fmt.Errorf("entity %d has no inventory", owner)
	}
	if i := slices.Index(inv.hotbar, -1); i != -1 {
		return x.BindHotbar(owner, i, item)
	}
	return nil
}

// Puts the item in the slot from into the slot to: into it if it's empty, onto a stack of the same kind as far as the
// stack's limit goes, leaving the rest behind, and swapped with whatever else is there. A stack emptied into another
// takes its place on the hotbar, and is removed at the end of the tick.
func (x *ECSContainer) MoveInventoryItem(owner common.EntityId, from, to int) error {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return fmt.Errorf("entity %d has no inventory", owner)
	}
	if from < 0 || from >= len(inv.slots) || to < 0 || to >= len(inv.slots) {
		return fmt.Errorf("can't move between slots %d and %d of an inventory of %d", from, to, len(inv.slots))
	}
	moving, there := inv.slots[from], inv.slots[to]
	if moving == -1 || from == to {
		return nil
	}

	if there != -1 {
		movingComp, _ := x.Items.Get(moving)
		thereComp, _ := x.Items.Get(there)
		if movingComp != nil && thereComp != nil && x.stacks(movingComp, thereComp) && thereComp.amount < x.stackLimit(thereComp) {
			moved := min(x.stackLimit(thereComp)-thereComp.amount, movingComp.amount)
			thereComp.amount += moved
			movingComp.amount -= moved
			if movingComp.amount > 0 {
				return nil
			}

			if i := inv.HotbarSlotOf(moving); i != -1 {
				inv.hotbar[i] = -1
				if inv.HotbarSlotOf(there) == -1 {
					inv.hotbar[i] = there
				}
			}
			x.takeOutOfInventory(owner, moving)
			x.SetDisabled(moving, true)
			x.ScheduleRemoveEntity(moving)
			return x.refreshHand(owner)
		}
	}

	inv.slots[from], inv.slots[to] = there, moving
	return nil
}

// Puts the item the hotbar slot points at in the owner's main hand, if it fits there, and enables it. Every other item
// in the inventory but what's in the off hand is disabled, so that what's carried but not in hand isn't drawn or used.
func (x *ECSContainer) SelectHotbarSlot(owner common.EntityId, hotbarSlot int) error {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return fmt.Errorf("entity %d has no inventory", owner)
	}
	if hotbarSlot < 0 || hotbarSlot >= len(inv.hotbar) {
		return fmt.Errorf("hotbar slot %d is outside the %d of entity %d's inventory", hotbarSlot, len(inv.hotbar), owner)
	}
	inv.selected = hotbarSlot
	return x.refreshHand(owner)
}

func (x *ECSContainer) refreshHand(owner common.EntityId) error {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return fmt.Errorf("entity %d has no inventory", owner)
	}
	inHand, _ := inv.HotbarItem(inv.selected)
	offHand := common.EntityId(-1)
	equipper, hasEquipper := x.Equippers.Get(owner)
	if hasEquipper {
		if e, ok := equipper.InSlot(Equip_OffHand); ok {
			offHand = e
		}
	}

	for _, item := range inv.slots {
		if item != -1 {
			x.SetDisabled(item, item != inHand && item != offHand)
		}
	}
	x.holdAt(inHand, -data.HeldItemOffsetPx)
	x.holdAt(offHand, data.HeldItemOffsetPx)

	if !hasEquipper {
		return nil
	}
	equipper.Unequip(Equip_MainHand)
	if inHand != -1 {
		if eq, ok := x.Equipments.Get(inHand); ok && eq.Fits(Equip_MainHand) {
			equipper.Equip(Equip_MainHand, inHand)
		}
	}
	return nil
}

// To the side of its holder's middle, which is left for the main hand
func (x *ECSContainer) holdAt(item common.EntityId, offsetX float64) {
	if t, ok := x.Transforms.Get(item); ok && item != -1 {
		t.SetLocalPos(utils.Vec2f{X: offsetX})
	}
}

// An item that stops being held by its owner leaves the owner's inventory with it, and any hotbar slot pointing at it
// points at nothing
func (x *ECSContainer) takeOutOfInventory(owner, item common.EntityId) {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return
	}
	if i := inv.slotOf(item); i != -1 {
		inv.slots[i] = -1
	}
	if i := inv.HotbarSlotOf(item); i != -1 {
		inv.hotbar[i] = -1
	}
}

// In milligrams
func (x *Inventory) MaxMass() int {
	return x.maxMass
}

// Of everything in the entity's inventory, in milligrams
func (x *ECSContainer) CarriedMass(e common.EntityId) int {
	inv, ok := x.Inventories.Get(e)
	if !ok {
		return 0
	}
	return x.carriedMass(inv)
}

func (x *ECSContainer) carriedMass(inv *Inventory) int {
	m := 0
	for i := range inv.Capacity() {
		if held, ok := x.heldItem(inv, i); ok {
			m += x.ItemMass(held)
		}
	}
	return m
}

// Never below 0, even for an inventory that was filled past its limit, which can only be done by putting things in it
// directly
func (x *ECSContainer) massRoom(inv *Inventory) int {
	return max(inv.maxMass-x.carriedMass(inv), 0)
}

func (x *Inventory) PickupsOffHotbar() bool {
	return x.pickupsOffHotbar
}

func (x *Inventory) SetPickupsOffHotbar(off bool) {
	x.pickupsOffHotbar = off
}

func (x Inventory) Copy() Inventory {
	return Inventory{slots: slices.Clone(x.slots), hotbar: slices.Clone(x.hotbar), selected: x.selected, pickupsOffHotbar: x.pickupsOffHotbar, maxMass: x.maxMass}
}

func (x Inventory) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Inventories.addComponent(e, x.Copy())
}

type inventoryDto struct {
	Slots            []common.EntityId
	Hotbar           []common.EntityId
	Selected         int
	PickupsOffHotbar bool
	MaxMass          int
}

func (x Inventory) toDto() (ComponentDto, error) {
	return &inventoryDto{Slots: slices.Clone(x.slots), Hotbar: slices.Clone(x.hotbar), Selected: x.selected, PickupsOffHotbar: x.pickupsOffHotbar, MaxMass: x.maxMass}, nil
}

func (x inventoryDto) toComponent() (Component, error) {
	return &Inventory{slots: slices.Clone(x.Slots), hotbar: slices.Clone(x.Hotbar), selected: x.Selected, pickupsOffHotbar: x.PickupsOffHotbar, maxMass: x.MaxMass}, nil
}
