package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

// Lets an entity be put in an inventory. Each item is its own entity, so two items of a kind can differ in any of
// their components.
type Item struct {
	// So that what was just dropped isn't picked straight back up by whoever dropped it
	pickupDelayMs int
	// 0 while nothing pulls it in
	pullSpeedPx float64
}

func NewItemComponent() *Item {
	return &Item{}
}

func (x *Item) CanBePickedUp() bool {
	return x.pickupDelayMs <= 0
}

func (x *Item) TickPickupDelay() {
	x.pickupDelayMs = max(x.pickupDelayMs-data.TickMs, 0)
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

// Puts the selected item on the ground where its owner stands, upright, and returns it
func (x *ECSContainer) DropSelectedItem(owner common.EntityId) (common.EntityId, bool, error) {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return -1, false, fmt.Errorf("entity %d has no inventory", owner)
	}

	item, ok := inv.Slot(inv.Selected())
	if !ok {
		return -1, false, nil
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

	angle := math.Atan2(aim.Y-from.Y, aim.X-from.X) + (2*x.Rng.Float64()-1)*data.ItemTossAngleVariationRad
	distance := data.ItemTossDistancePx * (1 + (2*x.Rng.Float64()-1)*data.ItemTossDistanceVariation)
	toss := utils.Vec2f{X: math.Cos(angle), Y: math.Sin(angle)}.Multiply(distance)

	x.AddComponent(item, NewTossComponent(toss, data.ItemTossDurationMs, data.ItemTossHeightPx))
}

// Puts the item lying on the ground into the lowest free slot of the collector's inventory, in its hand. False if it
// can't be picked up yet, or there's no free slot.
func (x *ECSContainer) PickUpItem(collector, item common.EntityId) (bool, error) {
	inv, ok := x.Inventories.Get(collector)
	if !ok {
		return false, fmt.Errorf("entity %d has no inventory", collector)
	}

	itemComp, ok := x.Items.Get(item)
	if !ok {
		return false, fmt.Errorf("entity %d isn't an item", item)
	}

	slot := inv.FirstFreeSlot()
	if !itemComp.CanBePickedUp() || x.Parent(item) != -1 || slot == -1 {
		return false, nil
	}

	x.EndToss(item)
	itemComp.StopPull()
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
	PickupDelayMs int
	PullSpeedPx   float64
}

func (x Item) toDto() (ComponentDto, error) {
	return &itemDto{PickupDelayMs: x.pickupDelayMs, PullSpeedPx: x.pullSpeedPx}, nil
}

func (x itemDto) toComponent() (Component, error) {
	return &Item{pickupDelayMs: x.PickupDelayMs, pullSpeedPx: x.PullSpeedPx}, nil
}
