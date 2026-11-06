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
	// So that it can be drawn and named without being brought back
	Name  string
	Image common.ImageAssetTag
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

// As many of the kind as fit on the tile and around it, topping up stacks like it before starting new ones on the
// closest empty tiles. How many were laid down; the rest have nowhere to go.
func (x *ECSContainer) LayDownKind(kind data.ItemEnum, count int, near utils.Vec2i) int {
	if _, ok := x.itemAtlas.Def(kind); !ok || count <= 0 {
		return 0
	}
	laid := 0
	// Topping up first, so that what's dropped joins what's already there rather than spreading out
	for _, fill := range []bool{true, false} {
		for _, t := range spillOrder(near) {
			if laid == count {
				return laid
			}
			on, taken := x.terrain.GroundStack(t)
			if taken != fill {
				continue
			}
			room := x.groundRoom(t, kind, false)
			if room <= 0 {
				continue
			}
			n := min(room, count-laid)
			if !x.terrain.SetGroundStack(t, GroundStack{Kind: kind, Count: on.Count + n}) {
				continue
			}
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
	unique := &UniqueItem{Root: item, Members: members, Name: itemComp.name}
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
		if inv.FirstFreeSlot() == -1 {
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
		return 1, x.giveItem(collector, stack.Unique.Root)
	}

	def, ok := x.itemAtlas.Def(stack.Kind)
	if !ok {
		return 0, fmt.Errorf("the stack on %v is of item %v, which isn't in the item atlas", t, stack.Kind)
	}
	taking := min(stack.Count, x.roomFor(inv, stack.Kind, def.maxStack))
	for left := taking; left > 0; {
		n := min(left, def.maxStack)
		item, err := x.SpawnItemStack(stack.Kind, n, pos)
		if err != nil {
			return taking - left, err
		}
		if err := x.giveItem(collector, item); err != nil {
			return taking - left, err
		}
		left -= n
	}
	x.terrain.SetGroundStack(t, GroundStack{Kind: stack.Kind, Count: stack.Count - taking})
	return taking, nil
}

// How many of the kind fit: what tops up its stacks, and a full stack in each free slot
func (x *ECSContainer) roomFor(inv *Inventory, kind data.ItemEnum, maxStack int) int {
	room := 0
	for i := range inv.Capacity() {
		held, ok := x.heldItem(inv, i)
		switch {
		case !ok && inv.slots[i] == -1:
			room += maxStack
		case ok && held.kind == kind && maxStack > 1:
			room += maxStack - held.count
		}
	}
	return room
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
