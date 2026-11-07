package ecs

import (
	"cmp"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"slices"
)

// A rectangle of tiles that holds what's put in it in its inventory, one slot for each tile, row by row, so that what's
// in a slot lies on its tile
type StorageArea struct {
	first, last utils.Vec2i
}

func NewStorageAreaComponent(first, last utils.Vec2i) *StorageArea {
	return &StorageArea{first: first, last: last}
}

func (x *StorageArea) Bounds() (first, last utils.Vec2i) {
	return x.first, x.last
}

func (x *StorageArea) Tiles() int {
	return (x.last.X - x.first.X + 1) * (x.last.Y - x.first.Y + 1)
}

func (x *StorageArea) Contains(t utils.Vec2i) bool {
	return t.X >= x.first.X && t.X <= x.last.X && t.Y >= x.first.Y && t.Y <= x.last.Y
}

func (x *StorageArea) touches(first, last utils.Vec2i) bool {
	return first.X <= x.last.X && last.X >= x.first.X && first.Y <= x.last.Y && last.Y >= x.first.Y
}

// The tile what's in the slot lies on
func (x *StorageArea) SlotTile(slot int) utils.Vec2i {
	w := x.last.X - x.first.X + 1
	return utils.Vec2i{X: x.first.X + slot%w, Y: x.first.Y + slot/w}
}

// Its tile closest to t, which is the one to reach to put something in it
func (x *StorageArea) ClosestTile(t utils.Vec2i) utils.Vec2i {
	return utils.Vec2i{X: min(max(t.X, x.first.X), x.last.X), Y: min(max(t.Y, x.first.Y), x.last.Y)}
}

func (x StorageArea) Copy() StorageArea {
	return StorageArea{first: x.first, last: x.last}
}

func (x StorageArea) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.StorageAreas.addComponent(e, x.Copy())
}

type storageAreaDto struct {
	First, Last utils.Vec2i
}

func (x StorageArea) toDto() (ComponentDto, error) {
	return &storageAreaDto{First: x.first, Last: x.last}, nil
}

func (x storageAreaDto) toComponent() (Component, error) {
	return &StorageArea{first: x.First, last: x.Last}, nil
}

// The storage area the tile is in
func (x *ECSContainer) StorageAt(t utils.Vec2i) (common.EntityId, bool) {
	for e, s := range Query1(&x.StorageAreas) {
		if s.Contains(t) {
			return e, true
		}
	}
	return -1, false
}

// Inside a colony's bounds, where colonists can get to it, on ground that can be walked on, without a site and
// without other storage
func (x *ECSContainer) CanPlaceStorage(first, last utils.Vec2i) bool {
	if first.X > last.X || first.Y > last.Y || !x.inAColony(first, last) {
		return false
	}
	for _, s := range Query1(&x.StorageAreas) {
		if s.touches(first, last) {
			return false
		}
	}
	for y := first.Y; y <= last.Y; y++ {
		for x2 := first.X; x2 <= last.X; x2++ {
			t := utils.Vec2i{X: x2, Y: y}
			if _, site := x.terrain.Site(t); site || x.terrain.WalkSpeedPercent(t) <= 0 {
				return false
			}
		}
	}
	return true
}

func (x *ECSContainer) inAColony(first, last utils.Vec2i) bool {
	for _, colony := range x.Colonies.GetEntities() {
		if x.InColony(colony, first) && x.InColony(colony, last) {
			return true
		}
	}
	return false
}

// Nothing if it doesn't fit, as CanPlaceStorage says
func (x *ECSContainer) PlaceStorage(first, last utils.Vec2i) (common.EntityId, bool, error) {
	if !x.CanPlaceStorage(first, last) {
		return -1, false, nil
	}
	area := NewStorageAreaComponent(first, last)
	center := utils.TilePosToWorldPos(first).Add(utils.TilePosToWorldPos(last)).Multiply(0.5)
	inv := NewInventoryComponent(area.Tiles(), 0)
	e, err := x.AddNewEntity(
		NewHierarchyComponent(),
		NewTransformComponent(center, 1, 0),
		NewPersistentComponent(),
		inv,
		area,
	)
	if err != nil {
		return -1, false, fmt.Errorf("error adding a storage area from %v to %v: %w", first, last, err)
	}
	return e, true, nil
}

// Every storage area the rectangle touches goes whole, with what was in it left lying on its tiles
func (x *ECSContainer) RemoveStorages(first, last utils.Vec2i) error {
	for e, s := range Query1(&x.StorageAreas) {
		if !s.touches(first, last) {
			continue
		}
		inv, ok := x.Inventories.Get(e)
		if !ok {
			return fmt.Errorf("storage area %d has no inventory", e)
		}
		for i := range inv.Capacity() {
			item, ok := inv.Slot(i)
			if !ok {
				continue
			}
			if _, err := x.LayDown(item, s.SlotTile(i)); err != nil {
				return fmt.Errorf("error laying down item %d of storage area %d: %w", item, e, err)
			}
		}
		x.ScheduleRemoveEntity(e)
	}
	return nil
}

// Whether there's room in it for at least some of the item
func (x *ECSContainer) HasRoomFor(owner, item common.EntityId) bool {
	inv, invOk := x.Inventories.Get(owner)
	itemComp, itemOk := x.Items.Get(item)
	if !invOk || !itemOk {
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

// Up to count of what's in the owner's slot goes in the storage area. How many went in.
func (x *ECSContainer) Store(owner common.EntityId, slot int, storage common.EntityId, count int) (int, error) {
	if !x.StorageAreas.HasComponent(storage) {
		return 0, nil
	}
	return x.MoveItems(owner, slot, storage, count)
}

// Up to count of the kind come out of the storage area into the owner's inventory, from its first slots first. How many
// came out.
func (x *ECSContainer) TakeFromStorage(owner, storage common.EntityId, kind data.ItemEnum, count int) (int, error) {
	inv, ok := x.Inventories.Get(storage)
	if !ok || !x.StorageAreas.HasComponent(storage) {
		return 0, nil
	}
	taken := 0
	for i := 0; i < inv.Capacity() && taken < count; i++ {
		if held, ok := x.heldItem(inv, i); !ok || held.kind != kind {
			continue
		}
		n, err := x.MoveItems(storage, i, owner, count-taken)
		if err != nil {
			return taken, err
		}
		if n == 0 {
			break
		}
		taken += n
	}
	return taken, nil
}

// Up to count of what's in from's slot goes into to's inventory, topping up stacks like it there first and then into
// its first free slot, as much as there's room for. Only what moves is split off a stack. How many moved.
func (x *ECSContainer) MoveItems(from common.EntityId, slot int, to common.EntityId, count int) (int, error) {
	fromInv, ok := x.Inventories.Get(from)
	if !ok {
		return 0, fmt.Errorf("entity %d has no inventory", from)
	}
	toInv, ok := x.Inventories.Get(to)
	if !ok {
		return 0, fmt.Errorf("entity %d has no inventory", to)
	}
	item, ok := fromInv.Slot(slot)
	if !ok || count <= 0 {
		return 0, nil
	}
	itemComp, ok := x.Items.Get(item)
	if !ok {
		return 0, fmt.Errorf("entity %d in slot %d of entity %d isn't an item", item, slot, from)
	}

	moving := min(count, itemComp.count)
	left := moving
	for i := 0; i < toInv.Capacity() && left > 0; i++ {
		held, ok := x.heldItem(toInv, i)
		if !ok || !x.stacks(held, itemComp) {
			continue
		}
		n := min(x.stackLimit(held)-held.count, left)
		held.count += n
		itemComp.count -= n
		left -= n
	}
	if free := toInv.FirstFreeSlot(); left > 0 && free != -1 {
		part := item
		if left < itemComp.count {
			pos, err := x.WorldPos(from)
			if err != nil {
				return 0, fmt.Errorf("error getting the world position of entity %d: %w", from, err)
			}
			if part, err = x.SpawnItemStack(itemComp.kind, left, pos); err != nil {
				return 0, fmt.Errorf("error splitting %d off item %d: %w", left, item, err)
			}
			itemComp.count -= left
		}
		if err := x.PutInInventory(to, free, part); err != nil {
			return 0, fmt.Errorf("error putting item %d in entity %d's inventory: %w", part, to, err)
		}
		left = 0
	}

	if itemComp.count == 0 {
		if err := x.Detach(item); err != nil {
			return 0, fmt.Errorf("error taking item %d out of entity %d: %w", item, from, err)
		}
		x.ScheduleRemoveEntity(item)
	}
	return moving - left, x.refreshHand(from)
}

// How many of each kind of item it holds, by kind
func (x *ECSContainer) StorageContents(storage common.EntityId) []ItemCost {
	inv, ok := x.Inventories.Get(storage)
	if !ok {
		return nil
	}
	contents := []ItemCost{}
	for i := range inv.Capacity() {
		held, ok := x.heldItem(inv, i)
		if !ok {
			continue
		}
		if j := slices.IndexFunc(contents, func(c ItemCost) bool { return c.Kind == held.kind }); j != -1 {
			contents[j].Count += held.count
		} else {
			contents = append(contents, ItemCost{Kind: held.kind, Count: held.count})
		}
	}
	slices.SortFunc(contents, func(a, b ItemCost) int { return cmp.Compare(a.Kind, b.Kind) })
	return contents
}
