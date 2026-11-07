package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

// What lies on a tile: a stack of a generic item's kind, or one unique item, kept as its entity's components until it's
// picked up, the way an unloaded chunk keeps its entities
type GroundStack struct {
	Kind   data.ItemEnum
	Count  int
	Unique *UniqueItem
}

type UniqueItem struct {
	Root    common.EntityId
	Members map[common.EntityId][]ComponentDto
	// So that it can be drawn, named and weighed without being brought back
	Name   string
	Image  common.ImageAssetTag
	Weight int
}

func (x *ECSContainer) GroundStackName(s GroundStack) string {
	if s.Unique != nil {
		return s.Unique.Name
	}
	if def, ok := x.itemAtlas.Def(s.Kind); ok {
		return def.name
	}
	return "?"
}

func (x *ECSContainer) GroundStackImage(s GroundStack) (common.ImageAssetTag, bool) {
	if s.Unique != nil {
		return s.Unique.Image, s.Unique.Image != ""
	}
	def, ok := x.itemAtlas.Def(s.Kind)
	return def.imageAsset, ok
}

// Whether the item can go on the tile: ground that can be walked on, with nothing on it, or with a stack of the same
// kind that has room
func (x *ECSContainer) groundRoom(t utils.Vec2i, kind data.ItemEnum, unique bool) int {
	if x.terrain.WalkSpeedPercent(t) <= 0 {
		return 0
	}
	on, taken := x.terrain.GroundStack(t)
	limit := 1
	if def, ok := x.itemAtlas.Def(kind); ok && !unique {
		limit = def.maxStack
	}
	switch {
	case !taken:
		return limit
	case unique || on.Unique != nil || on.Kind != kind:
		return 0
	default:
		return limit - on.Count
	}
}

// The tiles around near, closest first by rings, row by row within each, as far as items spill
func spillOrder(near utils.Vec2i) []utils.Vec2i {
	tiles := []utils.Vec2i{near}
	for r := 1; r <= data.GroundSpillRangeTiles; r++ {
		for y := near.Y - r; y <= near.Y+r; y++ {
			for x := near.X - r; x <= near.X+r; x++ {
				if max(abs(x-near.X), abs(y-near.Y)) == r {
					tiles = append(tiles, utils.Vec2i{X: x, Y: y})
				}
			}
		}
	}
	return tiles
}

// As many of the kind as fit on the tile and around it, closest first, each tile taking what fits on it, so that
// what's dropped stays as close as it can to where it was dropped. How many were laid down; the rest have nowhere to
// go.
func (x *ECSContainer) LayDownKind(kind data.ItemEnum, count int, near utils.Vec2i) int {
	if _, ok := x.itemAtlas.Def(kind); !ok || count <= 0 {
		return 0
	}
	laid := 0
	for _, t := range spillOrder(near) {
		if laid == count {
			break
		}
		room := x.groundRoom(t, kind, false)
		if room <= 0 {
			continue
		}
		on, _ := x.terrain.GroundStack(t)
		n := min(room, count-laid)
		if x.terrain.SetGroundStack(t, GroundStack{Kind: kind, Count: on.Count + n}) {
			laid += n
		}
	}
	return laid
}

// The item goes out of whatever holds it and onto the ground at the tile or as close to it as there's room. A
// generic item becomes part of the stacks there, and whatever doesn't fit stays where it was; a unique one goes whole
// onto the closest empty tile, kept as its components, or nowhere if there's none. Whether any of it went.
func (x *ECSContainer) LayDown(item common.EntityId, near utils.Vec2i) (bool, error) {
	itemComp, ok := x.Items.Get(item)
	if !ok {
		return false, fmt.Errorf("entity %d isn't an item", item)
	}

	if itemComp.kind != data.Item_None {
		laid := x.LayDownKind(itemComp.kind, itemComp.count, near)
		if laid == 0 {
			return false, nil
		}
		itemComp.count -= laid
		if itemComp.count > 0 {
			return true, nil
		}
		if err := x.letGo(item); err != nil {
			return true, err
		}
		x.removeEntity(item, false)
		return true, nil
	}

	tile, found := utils.Vec2i{}, false
	for _, t := range spillOrder(near) {
		if x.groundRoom(t, data.Item_None, true) > 0 {
			tile, found = t, true
			break
		}
	}
	if !found {
		return false, nil
	}
	if err := x.letGo(item); err != nil {
		return false, err
	}
	members, err := x.GetCompositionComponents(item)
	if err != nil {
		return false, fmt.Errorf("error keeping the components of item %d: %w", item, err)
	}
	unique := &UniqueItem{Root: item, Members: members, Name: itemComp.name, Weight: itemComp.weight}
	if spr, ok := x.Sprites.Get(item); ok {
		unique.Image = spr.ImageAsset()
	}
	if !x.terrain.SetGroundStack(tile, GroundStack{Count: 1, Unique: unique}) {
		return false, nil
	}
	// Right away rather than at the end of the tick, so that it can be picked back up in the same one
	x.removeEntity(item, false)
	return true, nil
}

// Out of its holder's hands and inventory, if it has one
func (x *ECSContainer) letGo(item common.EntityId) error {
	owner := x.Parent(item)
	if owner == -1 {
		return nil
	}
	if equipper, ok := x.Equippers.Get(owner); ok {
		for slot, eq := range equipper.equipment {
			if eq == item {
				equipper.Unequip(slot)
			}
		}
	}
	if err := x.Detach(item); err != nil {
		return fmt.Errorf("error taking item %d from entity %d: %w", item, owner, err)
	}
	x.SetDisabled(item, false)
	return x.refreshHand(owner)
}

// As much of what lies on the tile as there's room for goes into the collector's inventory, the rest staying. How
// many it took.
func (x *ECSContainer) PickUpFromGround(collector common.EntityId, t utils.Vec2i) (int, error) {
	stack, _ := x.terrain.GroundStack(t)
	return x.PickUpSomeFromGround(collector, t, stack.Count)
}

// Up to count of what lies on the tile, as much as there's room for. A unique item is all or nothing.
func (x *ECSContainer) PickUpSomeFromGround(collector common.EntityId, t utils.Vec2i, count int) (int, error) {
	stack, ok := x.terrain.GroundStack(t)
	inv, invOk := x.Inventories.Get(collector)
	if !ok || !invOk {
		return 0, nil
	}
	pos, err := x.WorldPos(collector)
	if err != nil {
		return 0, fmt.Errorf("error getting the position of entity %d: %w", collector, err)
	}

	if stack.Unique != nil {
		if inv.FirstFreeSlot() == -1 || stack.Unique.Weight > x.weightRoom(inv) {
			return 0, nil
		}
		for e, dtos := range stack.Unique.Members {
			comps := make([]Component, 0, len(dtos))
			for _, dto := range dtos {
				comp, err := DtoToComponent(dto)
				if err != nil {
					return 0, fmt.Errorf("error bringing back entity %d of the item on %v: %w", e, t, err)
				}
				comps = append(comps, comp)
			}
			if err := x.AddEntityWithId(e, comps...); err != nil {
				return 0, fmt.Errorf("error bringing back entity %d of the item on %v: %w", e, t, err)
			}
		}
		x.terrain.SetGroundStack(t, GroundStack{})
		return 1, x.GiveItem(collector, stack.Unique.Root)
	}

	def, ok := x.itemAtlas.Def(stack.Kind)
	if !ok {
		return 0, fmt.Errorf("the stack on %v is of item %v, which isn't in the item atlas", t, stack.Kind)
	}
	taking := min(count, stack.Count, x.roomFor(inv, stack.Kind, def))
	for left := taking; left > 0; {
		n := min(left, def.maxStack)
		item, err := x.SpawnItemStack(stack.Kind, n, pos)
		if err != nil {
			return taking - left, err
		}
		if err := x.GiveItem(collector, item); err != nil {
			return taking - left, err
		}
		left -= n
	}
	x.terrain.SetGroundStack(t, GroundStack{Kind: stack.Kind, Count: stack.Count - taking})
	return taking, nil
}

// Up to count of what's in the owner's slot goes onto the tile and only it, which has to be empty or hold a stack of
// the same kind with room. How many went.
func (x *ECSContainer) PutDownOn(owner common.EntityId, slot, count int, t utils.Vec2i) (int, error) {
	inv, ok := x.Inventories.Get(owner)
	if !ok {
		return 0, fmt.Errorf("entity %d has no inventory", owner)
	}
	item, ok := inv.Slot(slot)
	if !ok {
		return 0, nil
	}
	itemComp, ok := x.Items.Get(item)
	if !ok || itemComp.kind == data.Item_None {
		return 0, nil
	}
	n := min(count, itemComp.count, x.groundRoom(t, itemComp.kind, false))
	if n <= 0 {
		return 0, nil
	}
	on, _ := x.terrain.GroundStack(t)
	if !x.terrain.SetGroundStack(t, GroundStack{Kind: itemComp.kind, Count: on.Count + n}) {
		return 0, nil
	}
	itemComp.count -= n
	if itemComp.count > 0 {
		return n, nil
	}
	if err := x.letGo(item); err != nil {
		return n, err
	}
	x.removeEntity(item, false)
	return n, nil
}

// What lies on from goes onto to, as much as fits there: all of a unique item onto empty ground, or of a stack as much
// as tops up what's there or makes a new one. How many moved.
func (x *ECSContainer) MoveOnGround(from, to utils.Vec2i) int {
	stack, ok := x.terrain.GroundStack(from)
	if !ok || from == to {
		return 0
	}
	if stack.Unique != nil {
		if x.groundRoom(to, data.Item_None, true) <= 0 || !x.terrain.SetGroundStack(to, stack) {
			return 0
		}
		x.terrain.SetGroundStack(from, GroundStack{})
		return 1
	}
	n := min(stack.Count, x.groundRoom(to, stack.Kind, false))
	on, _ := x.terrain.GroundStack(to)
	if n <= 0 || !x.terrain.SetGroundStack(to, GroundStack{Kind: stack.Kind, Count: on.Count + n}) {
		return 0
	}
	x.terrain.SetGroundStack(from, GroundStack{Kind: stack.Kind, Count: stack.Count - n})
	return n
}

// How many of the kind fit on the tile: on empty ground that can be walked on a full stack, and on a stack of it
// what tops it up
func (x *ECSContainer) GroundRoomFor(t utils.Vec2i, kind data.ItemEnum) int {
	return x.groundRoom(t, kind, false)
}

// How many of the kind fit in the entity's inventory
func (x *ECSContainer) RoomFor(e common.EntityId, kind data.ItemEnum) int {
	inv, ok := x.Inventories.Get(e)
	def, defOk := x.itemAtlas.Def(kind)
	if !ok || !defOk {
		return 0
	}
	return x.roomFor(inv, kind, def)
}

// What tops up its stacks, and a full stack in each free slot, as far as what it can still carry goes
func (x *ECSContainer) roomFor(inv *Inventory, kind data.ItemEnum, def ItemDef) int {
	room := 0
	for i := range inv.Capacity() {
		held, ok := x.heldItem(inv, i)
		switch {
		case !ok && inv.slots[i] == -1:
			room += def.maxStack
		case ok && held.kind == kind && def.maxStack > 1:
			room += def.maxStack - held.count
		}
	}
	if def.weight > 0 {
		room = min(room, x.weightRoom(inv)/def.weight)
	}
	return room
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
