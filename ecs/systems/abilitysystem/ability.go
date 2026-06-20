package abilitysystem

import (
	"ebittest/ecs"
	"fmt"
	"slices"
)

func Tick(ecs *ecs.ECS) error {
	am := ecs.AbilitiesManager
	for _, aE := range slices.Clone(ecs.Abilities.GetEntities()) {
		err := am.TickAbilities(aE, ecs)
		if err != nil {
			return fmt.Errorf("error ticking ability of entity %d: %v", aE, err)
		}
	}

	eqm := ecs.EquipManager
	for _, eqE := range slices.Clone(ecs.Equipments.GetEntities()) {
		err := eqm.TickAbilities(eqE, ecs)
		if err != nil {
			return fmt.Errorf("error ticking equipment ability of entity %d: %v", eqE, err)
		}
	}

	drm := ecs.DeathrattleManager
	for _, e := range slices.Clone(ecs.Deathrattles.GetEntities()) {
		err := drm.TickAbilities(e, ecs)
		if err != nil {
			return fmt.Errorf("error ticking deathrattle ability of entity %d: %v", e, err)
		}
	}

	return nil
}
