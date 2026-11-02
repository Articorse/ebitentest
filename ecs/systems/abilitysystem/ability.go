package abilitysystem

import (
	"ebittest/ecs"
	"fmt"
	"slices"
)

func Tick(ecsContainer *ecs.ECSContainer) error {
	for _, aE := range slices.Clone(ecsContainer.Abilities.GetEntities()) {
		if ecsContainer.IsDisabled(aE) {
			continue
		}
		err := ecsContainer.TickAbilities(aE)
		if err != nil {
			return fmt.Errorf("error ticking ability of entity %d: %v", aE, err)
		}
	}

	for _, eqE := range slices.Clone(ecsContainer.Equipments.GetEntities()) {
		if ecsContainer.IsDisabled(eqE) {
			continue
		}
		err := ecsContainer.TickEquipmentAbilities(eqE)
		if err != nil {
			return fmt.Errorf("error ticking equipment ability of entity %d: %v", eqE, err)
		}
	}

	for _, e := range slices.Clone(ecsContainer.Deathrattles.GetEntities()) {
		if ecsContainer.IsDisabled(e) {
			continue
		}
		err := ecsContainer.TickDeathrattle(e)
		if err != nil {
			return fmt.Errorf("error ticking deathrattle ability of entity %d: %v", e, err)
		}
	}

	return nil
}
