package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

type ItemKindEnum uint16

// The other kinds are defined by the content that registers them. Items without a kind never stack, and can't be
// spawned by kind.
const ItemKind_None ItemKindEnum = 0

// Spawns one of an item kind lying at pos
type ItemSpawnFunc func(pos utils.Vec2f, ecsContainer *ECSContainer) (common.EntityId, error)

// Lets an entity be put in an inventory. Each item is its own entity, so two items of a kind can differ in any of
// their components. A stack of an item is still a single entity, with a count.
type Item struct {
	kind     ItemKindEnum
	count    int
	maxStack int

	// So that what was just dropped isn't picked straight back up by whoever dropped it
	pickupDelayMs int
	// 0 while nothing pulls it in
	pullSpeedPx float64
}

func NewItemComponent() *Item {
	return &Item{count: 1, maxStack: 1}
}

func NewStackableItemComponent(kind ItemKindEnum, maxStack int) *Item {
	return &Item{kind: kind, count: 1, maxStack: maxStack}
}

func (x *Item) Kind() ItemKindEnum {
	return x.kind
}

func (x *Item) Count() int {
	return x.count
}

func (x *Item) stacksWith(other *Item) bool {
	return x.kind != ItemKind_None && x.kind == other.kind && x.maxStack > 1
}

// An empty stack is waiting to be removed
func (x *Item) CanBePickedUp() bool {
	return x.pickupDelayMs <= 0 && x.count > 0
}

func (x *Item) TickPickupDelay() {
	x.pickupDelayMs = max(x.pickupDelayMs-data.TickMs, 0)
}

func (x *ECSContainer) SpawnItem(kind ItemKindEnum, pos utils.Vec2f) (common.EntityId, error) {
	spawn, ok := x.itemSpawnFuncs[kind]
	if !ok {
		return -1, fmt.Errorf("no item of kind %v is registered", kind)
	}
	return spawn(pos, x)
}

// Pulls the loose item a tick towards target, faster each tick it's pulled, without passing it or going through
// terrain
func (x *ECSContainer) PullItem(e common.EntityId, target utils.Vec2f) {
	item, itemOk := x.Items.Get(e)
	t, traOk := x.Transforms.Get(e)
	if !itemOk || !traOk || x.Parent(e) != -1 {
		return
	}

	if item.pullSpeedPx == 0 {
		item.pullSpeedPx = data.ItemPullStartSpeedPx
	} else {
		item.pullSpeedPx += data.ItemPullAccelerationPx
	}

	toTarget := target.Subtract(t.LocalPos())
	if distance := toTarget.Length(); distance > 0 {
		moveOnGround(t, toTarget.Multiply(min(item.pullSpeedPx, distance)/distance), x.terrain)
	}
}

func (x *Item) StopPull() {
	x.pullSpeedPx = 0
}

// Puts the selected item, or one of a stack, on the ground where its owner stands, upright, tossed towards the
// owner's aim, and returns it
func (x *ECSContainer) DropSelectedItem(owner common.EntityId) (common.EntityId, bool, error) {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return -1, false, fmt.Errorf("entity %d has no inventory", owner)
	}

	item, ok := inv.Slot(inv.Selected())
	if !ok {
		return -1, false, nil
	}

	if itemComp, ok := x.Items.Get(item); ok && itemComp.count > 1 {
		return x.dropOneOfStack(owner, item, itemComp)
	}

	if equipper, ok := x.Equippers.Get(owner); ok {
		for slot, eq := range equipper.equipment {
			if eq == item {
				equipper.Unequip(slot)
			}
		}
	}

	// Taking it out of the owner's hand also takes it out of the inventory
	if err := x.Detach(item); err != nil {
		return -1, false, fmt.Errorf("error detaching item %d from entity %d: %w", item, owner, err)
	}
	if t, ok := x.Transforms.Get(item); ok {
		t.SetLocalRotation(0)
	}
	x.SetDisabled(item, false)

	if itemComp, ok := x.Items.Get(item); ok {
		itemComp.pickupDelayMs = data.ItemPickupDelayMs
	}

	x.tossTowardsAim(owner, item)

	return item, true, nil
}

func (x *ECSContainer) dropOneOfStack(owner, stack common.EntityId, stackComp *Item) (common.EntityId, bool, error) {
	pos, err := x.WorldPos(owner)
	if err != nil {
		return -1, false, fmt.Errorf("error getting world position of entity %d: %w", owner, err)
	}

	one, err := x.SpawnItem(stackComp.kind, pos)
	if err != nil {
		return -1, false, fmt.Errorf("error splitting one off stack %d: %w", stack, err)
	}
	stackComp.count--

	if oneComp, ok := x.Items.Get(one); ok {
		oneComp.pickupDelayMs = data.ItemPickupDelayMs
	}
	x.tossTowardsAim(owner, one)

	return one, true, nil
}

// Slightly off, in direction and distance, so that dropping things in a row doesn't stack them
func (x *ECSContainer) tossTowardsAim(owner, item common.EntityId) {
	aim, ok := x.AimOf(owner)
	if !ok {
		return
	}
	from, err := x.WorldPos(item)
	if err != nil || aim == from {
		return
	}

	x.tossItem(item, math.Atan2(aim.Y-from.Y, aim.X-from.X)+(2*x.Rng.Float64()-1)*data.ItemTossAngleVariationRad)
}

func (x *ECSContainer) tossItem(item common.EntityId, angle float64) {
	distance := data.ItemTossDistancePx * (1 + (2*x.Rng.Float64()-1)*data.ItemTossDistanceVariation)
	toss := utils.Vec2f{X: math.Cos(angle), Y: math.Sin(angle)}.Multiply(distance)
	x.AddComponent(item, NewTossComponent(toss, data.ItemTossDurationMs, data.ItemTossHeightPx))
}

// Whether the collector has room for at least some of the loose item, which isn't flying through the air or waiting
// out its pickup delay
func (x *ECSContainer) CanPickUp(collector, item common.EntityId) bool {
	inv, invOk := x.Inventories.Get(collector)
	itemComp, itemOk := x.Items.Get(item)
	if !invOk || !itemOk || !itemComp.CanBePickedUp() || x.Parent(item) != -1 || x.Tosses.HasComponent(item) {
		return false
	}

	if inv.FirstFreeSlot() != -1 {
		return true
	}
	for i := range inv.Capacity() {
		if held, ok := x.heldItem(inv, i); ok && held.stacksWith(itemComp) && held.count < held.maxStack {
			return true
		}
	}
	return false
}

func (x *ECSContainer) heldItem(inv *Inventory, slot int) (*Item, bool) {
	e, ok := inv.Slot(slot)
	if !ok {
		return nil, false
	}
	return x.Items.Get(e)
}

// Tops up the collector's stacks of the item first, in slot order, and puts what's left into the lowest free slot, in
// its hand. An item that's used up by topping up is removed at the end of the tick. False if the collector can't pick
// it up.
func (x *ECSContainer) PickUpItem(collector, item common.EntityId) (bool, error) {
	if !x.CanPickUp(collector, item) {
		return false, nil
	}
	inv, _ := x.Inventories.Get(collector)
	itemComp, _ := x.Items.Get(item)

	x.EndToss(item)
	itemComp.StopPull()

	for i := range inv.Capacity() {
		held, ok := x.heldItem(inv, i)
		if !ok || !held.stacksWith(itemComp) {
			continue
		}
		moved := min(held.maxStack-held.count, itemComp.count)
		held.count += moved
		itemComp.count -= moved
	}
	if itemComp.count == 0 {
		x.ScheduleRemoveEntity(item)
		return true, nil
	}

	slot := inv.FirstFreeSlot()
	if slot == -1 {
		return true, nil
	}
	if err := x.PutInInventory(collector, slot, item); err != nil {
		return false, err
	}
	if t, ok := x.Transforms.Get(item); ok {
		t.SetLocalPos(utils.Vec2f{})
		t.SetLocalRotation(0)
	}

	// Selecting again enables and equips the item if it went into the selected slot, and disables it otherwise
	if err := x.SelectInventorySlot(collector, inv.Selected()); err != nil {
		return false, err
	}

	return true, nil
}

func (x Item) Copy() Item {
	return x
}

func (x Item) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Items.addComponent(e, x.Copy())
}

type itemDto struct {
	Kind          ItemKindEnum
	Count         int
	MaxStack      int
	PickupDelayMs int
	PullSpeedPx   float64
}

func (x Item) toDto() (ComponentDto, error) {
	return &itemDto{
		Kind:          x.kind,
		Count:         x.count,
		MaxStack:      x.maxStack,
		PickupDelayMs: x.pickupDelayMs,
		PullSpeedPx:   x.pullSpeedPx,
	}, nil
}

func (x itemDto) toComponent() (Component, error) {
	return &Item{
		kind:          x.Kind,
		count:         x.Count,
		maxStack:      x.MaxStack,
		pickupDelayMs: x.PickupDelayMs,
		pullSpeedPx:   x.PullSpeedPx,
	}, nil
}
