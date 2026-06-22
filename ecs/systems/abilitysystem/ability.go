package abilitysystem

import (
	"ebittest/ecs"
	"fmt"
	"slices"
)

func Tick(ecsContainer *ecs.ECSContainer) error {
	am := ecsContainer.AbilitiesManager
	for _, aE := range slices.Clone(ecsContainer.Abilities.GetEntities()) {
		err := am.TickAbilities(aE, ecsContainer)
		if err != nil {
			return fmt.Errorf("error ticking ability of entity %d: %v", aE, err)
		}
	}

	eqm := ecsContainer.EquipManager
	for _, eqE := range slices.Clone(ecsContainer.Equipments.GetEntities()) {
		err := eqm.TickAbilities(eqE, ecsContainer)
		if err != nil {
			return fmt.Errorf("error ticking equipment ability of entity %d: %v", eqE, err)
		}
	}

	drm := ecsContainer.DeathrattleManager
	for _, e := range slices.Clone(ecsContainer.Deathrattles.GetEntities()) {
		err := drm.TickAbilities(e, ecsContainer)
		if err != nil {
			return fmt.Errorf("error ticking deathrattle ability of entity %d: %v", e, err)
		}
	}

	return nil
}
