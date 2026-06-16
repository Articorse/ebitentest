package abilitysystem

import (
	"ebittest/ecs"
	"fmt"
	"slices"
)

func Tick(world *ecs.World) error {
	am := ecs.AbilitiesManager{}
	for _, aE := range slices.Clone(world.Abilities.GetEntities()) {
		err := am.TickAbilities(aE, world)
		if err != nil {
			return fmt.Errorf("error ticking ability of entity %d: %v", aE, err)
		}
	}

	eqm := ecs.EquipManager{}
	for _, eqE := range slices.Clone(world.Equipments.GetEntities()) {
		err := eqm.TickAbilities(eqE, world)
		if err != nil {
			return fmt.Errorf("error ticking equipment ability of entity %d: %v", eqE, err)
		}
	}

	return nil
}
