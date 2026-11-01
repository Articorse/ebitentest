package abilitysystem

import (
	"ebittest/ecs"
	"fmt"
	"slices"
)

func Tick(ecsContainer *ecs.ECSContainer) error {
	for _, aE := range slices.Clone(ecsContainer.Abilities.GetEntities()) {
		err := ecsContainer.TickAbilities(aE)
		if err != nil {
			return fmt.Errorf("error ticking ability of entity %d: %v", aE, err)
		}
	}

	for _, eqE := range slices.Clone(ecsContainer.Equipments.GetEntities()) {
		err := ecsContainer.TickEquipmentAbilities(eqE)
		if err != nil {
			return fmt.Errorf("error ticking equipment ability of entity %d: %v", eqE, err)
		}
	}

	for _, e := range slices.Clone(ecsContainer.Deathrattles.GetEntities()) {
		err := ecsContainer.TickDeathrattle(e)
		if err != nil {
			return fmt.Errorf("error ticking deathrattle ability of entity %d: %v", e, err)
		}
	}

	return nil
}
