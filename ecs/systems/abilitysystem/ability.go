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

	return nil
}
