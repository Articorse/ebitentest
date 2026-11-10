package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
)

// Lets an entity be put in an inventory. A unique item, such as a tool, is its own entity, so two of them can differ
// in any of their components. A generic item, such as wood, is only its kind and amount, so a stack of it is a single
// entity, and everything else about it comes from the item atlas.
type Item struct {
	kind data.ItemEnum
	// How many of a countable item, the milligrams of anything else, and 1 for a unique item
	amount int
	// A unique item's, since it has no kind to look it up by
	name string
	// A unique item's, in milligrams, for the same reason
	mass int
}

func NewItemComponent() *Item {
	return &Item{amount: 1}
}

// A unique item with a name to show, weighing massMg
func NewNamedItemComponent(name string, massMg int) *Item {
	return &Item{amount: 1, name: name, mass: massMg}
}

func (x *Item) Kind() data.ItemEnum {
	return x.kind
}

func (x *Item) Amount() int {
	return x.amount
}

// 1 for unique items, which the atlas never has, and generic ones whose kind isn't in it
func (x *ECSContainer) stackLimit(item *Item) int {
	if def, ok := x.itemAtlas.Def(item.kind); ok {
		return def.maxStack
	}
	return 1
}

// In milligrams: a unique item's own, a generic one's by its kind, and 0 for a kind that isn't in the atlas
func (x *ECSContainer) ItemMass(item *Item) int {
	if item.kind == data.Item_None {
		return item.mass
	}
	def, _ := x.itemAtlas.Def(item.kind)
	return def.Mass(item.amount)
}

// What's shown of how much of it there is: a countable item's count ahead of its mass, and anything else's mass
func (x *ECSContainer) ItemAmountText(item *Item) string {
	if item.kind == data.Item_None {
		return utils.FormatMass(item.mass)
	}
	def, _ := x.itemAtlas.Def(item.kind)
	return def.AmountText(item.amount)
}

// A unique item's own name, and a generic one's kind's
func (x *ECSContainer) ItemName(item *Item) string {
	if item.kind == data.Item_None {
		return item.name
	}
	if def, ok := x.itemAtlas.Def(item.kind); ok {
		return def.name
	}
	return "?"
}

func (x *ECSContainer) stacks(a, b *Item) bool {
	return a.kind == b.kind && x.stackLimit(a) > 1
}

// One of a countable item, or a kilogram of anything else, lying at pos
func (x *ECSContainer) SpawnItem(kind data.ItemEnum, pos utils.Vec2f) (common.EntityId, error) {
	return x.SpawnItemStack(kind, x.oneOf(&Item{kind: kind}), pos)
}

// A stack of a generic item, lying at pos
func (x *ECSContainer) SpawnItemStack(kind data.ItemEnum, amount int, pos utils.Vec2f) (common.EntityId, error) {
	def, ok := x.itemAtlas.Def(kind)
	if !ok {
		return -1, fmt.Errorf("item %v isn't in the item atlas", kind)
	}
	if amount <= 0 || amount > def.maxStack {
		return -1, fmt.Errorf("a stack of item %v holds 1 to %d, not %d", kind, def.maxStack, amount)
	}

	return x.AddNewEntity(
		NewHierarchyComponent(),
		NewTransformComponent(pos, 1, 0),
		&Item{kind: kind, amount: amount},
		NewPersistentComponent(),
	)
}

// One of a countable stack in hand, a kilogram of anything else or all of it if there's less, or the whole of a unique
// item, goes on the ground at the tile or as close to it as there's room. Whether any went.
func (x *ECSContainer) DropSelected(owner common.EntityId, t utils.Vec2i) (bool, error) {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return false, fmt.Errorf("entity %d has no inventory", owner)
	}
	item, ok := inv.HotbarItem(inv.Selected())
	if !ok {
		return false, nil
	}
	itemComp, ok := x.Items.Get(item)
	if !ok || itemComp.kind == data.Item_None {
		return x.LayDown(item, t)
	}
	one := x.oneOf(itemComp)
	if itemComp.amount <= one {
		return x.LayDown(item, t)
	}
	laid := x.LayDownKind(itemComp.kind, one, t)
	itemComp.amount -= laid
	return laid > 0, nil
}

// What's handed over at a time where it isn't all of it: one of a countable item, and a kilogram of anything else
func (x *ECSContainer) oneOf(item *Item) int {
	if def, ok := x.itemAtlas.Def(item.kind); ok && !def.countable {
		return data.MilligramsPerKg
	}
	return 1
}

func (x *ECSContainer) heldItem(inv *Inventory, slot int) (*Item, bool) {
	e, ok := inv.Slot(slot)
	if !ok {
		return nil, false
	}
	return x.Items.Get(e)
}

// Into the collector's inventory, topping up stacks like it first, then into the first free slot, bound to a free
// hotbar slot. What there's no room for is left as it is.
func (x *ECSContainer) GiveItem(collector, item common.EntityId) error {
	inv, ok := x.Inventories.Get(collector)
	if !ok {
		return fmt.Errorf("entity %d has no inventory", collector)
	}
	itemComp, ok := x.Items.Get(item)
	if !ok {
		return fmt.Errorf("entity %d isn't an item", item)
	}

	for i := range inv.Capacity() {
		held, ok := x.heldItem(inv, i)
		if !ok || !x.stacks(held, itemComp) {
			continue
		}
		moved := min(x.stackLimit(held)-held.amount, itemComp.amount)
		held.amount += moved
		itemComp.amount -= moved
	}
	if itemComp.amount == 0 {
		x.ScheduleRemoveEntity(item)
		return nil
	}

	slot := inv.FirstFreeSlot()
	if slot == -1 {
		return nil
	}
	if err := x.PutInInventory(collector, slot, item); err != nil {
		return err
	}
	if t, ok := x.Transforms.Get(item); ok {
		t.SetLocalPos(utils.Vec2f{})
		t.SetLocalRotation(0)
	}

	if inv.PickupsOffHotbar() {
		return nil
	}
	return x.bindToFreeHotbarSlot(collector, item)
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
	Kind   data.ItemEnum
	Amount int
	Name   string
	Mass   int
}

func (x Item) toDto() (ComponentDto, error) {
	return &itemDto{
		Kind:   x.kind,
		Amount: x.amount,
		Name:   x.name,
		Mass:   x.mass,
	}, nil
}

func (x itemDto) toComponent() (Component, error) {
	return &Item{
		kind:   x.Kind,
		amount: x.Amount,
		name:   x.Name,
		mass:   x.Mass,
	}, nil
}

// Uses up one of a countable item
func (x *ECSContainer) ConsumeItem(item common.EntityId) error {
	return x.UseUp(item, 1)
}

// Uses up an amount of the item. What uses up the last of it leaves its holder's hand and inventory right away, rather
// than when it's removed at the end of the tick, so that nothing can be put into its empty stack in the meantime.
func (x *ECSContainer) UseUp(item common.EntityId, amount int) error {
	itemComp, ok := x.Items.Get(item)
	if !ok {
		return fmt.Errorf("entity %d isn't an item", item)
	}
	if itemComp.amount < amount || amount <= 0 {
		return fmt.Errorf("item %d has %d to use up %d of", item, itemComp.amount, amount)
	}

	itemComp.amount -= amount
	if itemComp.amount > 0 {
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
