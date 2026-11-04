package ecs

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// Returns every broken reference between live entities, and every image they draw that the assets don't have. None of
// these should ever happen, so a non-nil result points at a bug rather than at a state the game should handle.
func (x *ECSContainer) CheckConsistency() error {
	var errs error

	for _, e := range x.Equippers.GetEntities() {
		equipperComp, _ := x.Equippers.Get(e)

		for _, slot := range slices.Sorted(maps.Keys(equipperComp.equipment)) {
			eq := equipperComp.equipment[slot]

			switch {
			case !x.HasEntity(eq):
				errs = errors.Join(errs, fmt.Errorf("equipper %d references missing entity %d in slot %v", e, eq, slot))
			case !x.Equipments.HasComponent(eq):
				errs = errors.Join(errs, fmt.Errorf("equipper %d references entity %d in slot %v, which has no equipment component", e, eq, slot))
			case x.Parent(eq) != e || x.LinkKind(eq) != Link_Held:
				errs = errors.Join(errs, fmt.Errorf("equipment %d in slot %v of equipper %d isn't held by it", eq, slot, e))
			}
		}
	}

	for e, inv := range Query1(&x.Inventories) {
		if inv.selected < 0 || inv.selected >= len(inv.slots) {
			errs = errors.Join(errs, fmt.Errorf("inventory %d has slot %d selected, outside its %d slots", e, inv.selected, len(inv.slots)))
		}

		for slot, item := range inv.slots {
			switch {
			case item == -1:
			case !x.HasEntity(item):
				errs = errors.Join(errs, fmt.Errorf("inventory %d references missing entity %d in slot %d", e, item, slot))
			case !x.Items.HasComponent(item):
				errs = errors.Join(errs, fmt.Errorf("inventory %d references entity %d in slot %d, which isn't an item", e, item, slot))
			case x.Parent(item) != e || x.LinkKind(item) != Link_Held:
				errs = errors.Join(errs, fmt.Errorf("item %d in slot %d of inventory %d isn't held by it", item, slot, e))
			case slices.Index(inv.slots, item) != slot:
				errs = errors.Join(errs, fmt.Errorf("item %d is in inventory %d more than once", item, e))
			}
		}
	}

	for _, e := range x.Hierarchies.GetEntities() {
		p := x.Parent(e)
		if p != -1 && !slices.Contains(x.Children(p), e) {
			errs = errors.Join(errs, fmt.Errorf("entity %d has parent %d, which doesn't list it as a child", e, p))
		}

		children := x.Children(e)
		for i, c := range children {
			if slices.Contains(children[:i], c) {
				errs = errors.Join(errs, fmt.Errorf("entity %d lists child %d more than once", e, c))
				continue
			}
			if x.Parent(c) != e {
				errs = errors.Join(errs, fmt.Errorf("entity %d lists child %d, whose parent is %d", e, c, x.Parent(c)))
			}
		}
	}

	for _, e := range x.Tethers.GetEntities() {
		tetherComp, _ := x.Tethers.Get(e)
		if !x.HasEntity(tetherComp.anchor) {
			errs = errors.Join(errs, fmt.Errorf("entity %d is tethered to missing entity %d", e, tetherComp.anchor))
		} else if !x.ChunkLoaders.HasComponent(tetherComp.anchor) {
			errs = errors.Join(errs, fmt.Errorf("entity %d is tethered to entity %d, which isn't a chunkloader", e, tetherComp.anchor))
		}
	}

	errs = errors.Join(errs, x.checkImages())

	return errs
}
