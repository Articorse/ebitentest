package ecs

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// Returns every broken reference between live entities. None of these should ever happen,
// so a non-nil result points at a bug rather than at a state the game should handle.
func (x *ECSContainer) CheckConsistency() error {
	hm := hierarchyManager{}
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
			case hm.GetParent(eq, x) != e || hm.GetLinkKind(eq, x) != Link_Held:
				errs = errors.Join(errs, fmt.Errorf("equipment %d in slot %v of equipper %d isn't held by it", eq, slot, e))
			}
		}
	}

	for _, e := range x.Hierarchies.GetEntities() {
		p := hm.GetParent(e, x)
		if p != -1 && !slices.Contains(hm.GetChildren(p, x), e) {
			errs = errors.Join(errs, fmt.Errorf("entity %d has parent %d, which doesn't list it as a child", e, p))
		}

		children := hm.GetChildren(e, x)
		for i, c := range children {
			if slices.Contains(children[:i], c) {
				errs = errors.Join(errs, fmt.Errorf("entity %d lists child %d more than once", e, c))
				continue
			}
			if hm.GetParent(c, x) != e {
				errs = errors.Join(errs, fmt.Errorf("entity %d lists child %d, whose parent is %d", e, c, hm.GetParent(c, x)))
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

	return errs
}
