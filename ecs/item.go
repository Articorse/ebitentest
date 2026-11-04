package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
	"math"
)

// Lets an entity be put in an inventory. A unique item, such as a tool, is its own entity, so two of them can differ
// in any of their components. A generic item, such as planks, is only its kind and count, so a stack of it is a
// single entity, and everything else about it comes from the item atlas.
type Item struct {
	kind  data.ItemEnum
	count int

	// So that what was just dropped isn't picked straight back up by whoever dropped it
	pickupDelayMs int
	// 0 while nothing pulls it in
	pullSpeedPx float64
}

func NewItemComponent() *Item {
	return &Item{count: 1}
}

func (x *Item) Kind() data.ItemEnum {
	return x.kind
}

func (x *Item) Count() int {
	return x.count
}

// 1 for unique items, which the atlas never has, and generic ones whose kind isn't in it
func (x *ECSContainer) stackLimit(item *Item) int {
	if def, ok := x.itemAtlas.Def(item.kind); ok {
		return def.maxStack
	}
	return 1
}

func (x *ECSContainer) stacks(a, b *Item) bool {
	return a.kind == b.kind && x.stackLimit(a) > 1
}

// An empty stack is waiting to be removed
func (x *Item) CanBePickedUp() bool {
	return x.pickupDelayMs <= 0 && x.count > 0
}

func (x *Item) TickPickupDelay() {
	x.pickupDelayMs = max(x.pickupDelayMs-data.TickMs, 0)
}

// One of a generic item, lying at pos
func (x *ECSContainer) SpawnItem(kind data.ItemEnum, pos utils.Vec2f) (common.EntityId, error) {
	return x.SpawnItemStack(kind, 1, pos)
}

// A stack of a generic item, lying at pos
func (x *ECSContainer) SpawnItemStack(kind data.ItemEnum, count int, pos utils.Vec2f) (common.EntityId, error) {
	def, ok := x.itemAtlas.Def(kind)
	if !ok {
		return -1, fmt.Errorf("item %v isn't in the item atlas", kind)
	}
	if count <= 0 || count > def.maxStack {
		return -1, fmt.Errorf("a stack of item %v holds 1 to %d, not %d", kind, def.maxStack, count)
	}

	return x.AddNewEntity(
		NewHierarchyComponent(),
		NewTransformComponent(pos, 1, 0),
		&Item{kind: kind, count: count},
		NewPersistentComponent(),
	)
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

// Puts the item in hand, or one of a stack, on the ground where its owner stands, upright, tossed towards the owner's
// aim, and returns it
func (x *ECSContainer) DropSelectedItem(owner common.EntityId) (common.EntityId, bool, error) {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return -1, false, fmt.Errorf("entity %d has no inventory", owner)
	}

	item, ok := inv.HotbarItem(inv.Selected())
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
		if held, ok := x.heldItem(inv, i); ok && x.stacks(held, itemComp) && held.count < x.stackLimit(held) {
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

// Tops up the collector's stacks of the item first, in slot order, and puts what's left into the lowest free slot and
// on the first hotbar slot that points at nothing. An item that's used up by topping up is removed at the end of the
// tick. False if the collector can't pick it up.
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
		if !ok || !x.stacks(held, itemComp) {
			continue
		}
		moved := min(x.stackLimit(held)-held.count, itemComp.count)
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

	if err := x.bindToFreeHotbarSlot(collector, item); err != nil {
		return false, err
	}

	return true, nil
}

func (x Item) Copy() Item {
	return x
}

// A generic item gets its sprite, and its animation, equipment and light if it has them, from the atlas whenever it's
// added, spawned or loaded, since those aren't saved
func (x Item) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Items.addComponent(e, x.Copy())

	if x.kind == data.Item_None {
		return
	}
	def, ok := ecsContainer.itemAtlas.Def(x.kind)
	if !ok {
		log.Printf("item %d is of kind %v, which isn't in the item atlas", e, x.kind)
		return
	}

	if eq, ok := ecsContainer.itemEquipment(def); ok {
		ecsContainer.AddComponent(e, eq)
	}

	if def.lightLevel > 0 {
		light, err := NewLightEmitterComponent(def.lightLevel, def.lightRangeTiles)
		if err != nil {
			log.Printf("error creating the light of item %d: %v", e, err)
		} else {
			ecsContainer.AddComponent(e, light)
		}
	}

	spr, err := NewSpriteComponent(def.imageAsset, data.ItemSpriteLayer, false)
	if err != nil {
		log.Printf("error creating the sprite of item %d: %v", e, err)
		return
	}
	ecsContainer.AddComponent(e, spr)

	if len(def.animation) > 0 {
		anim, err := NewAnimationComponent(def.imageAsset, map[AnimationState][]AnimationFrame{Anim_Idle: def.animation})
		if err != nil {
			log.Printf("error creating the animation of item %d: %v", e, err)
			return
		}
		ecsContainer.AddComponent(e, anim)
	}
}

func (x *ECSContainer) isGenericItem(e common.EntityId) bool {
	item, ok := x.Items.Get(e)
	return ok && item.kind != data.Item_None
}

type itemDto struct {
	Kind          data.ItemEnum
	Count         int
	PickupDelayMs int
	PullSpeedPx   float64
}

func (x Item) toDto() (ComponentDto, error) {
	return &itemDto{
		Kind:          x.kind,
		Count:         x.count,
		PickupDelayMs: x.pickupDelayMs,
		PullSpeedPx:   x.pullSpeedPx,
	}, nil
}

func (x itemDto) toComponent() (Component, error) {
	return &Item{
		kind:          x.Kind,
		count:         x.Count,
		pickupDelayMs: x.PickupDelayMs,
		pullSpeedPx:   x.PullSpeedPx,
	}, nil
}

// Uses up one of the item. The last one leaves its holder's hand and inventory right away, rather than when it's
// removed at the end of the tick, so that nothing can be put into its empty stack in the meantime.
func (x *ECSContainer) ConsumeItem(item common.EntityId) error {
	itemComp, ok := x.Items.Get(item)
	if !ok {
		return fmt.Errorf("entity %d isn't an item", item)
	}
	if itemComp.count <= 0 {
		return fmt.Errorf("item %d is already used up", item)
	}

	itemComp.count--
	if itemComp.count > 0 {
		return nil
	}

	if holder := x.Parent(item); holder != -1 {
		if equipper, ok := x.Equippers.Get(holder); ok {
			for slot, eq := range equipper.equipment {
				if eq == item {
					equipper.Unequip(slot)
				}
			}
		}
		x.takeOutOfInventory(holder, item)
	}
	x.SetDisabled(item, true)
	x.ScheduleRemoveEntity(item)

	return nil
}
