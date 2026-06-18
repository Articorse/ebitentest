package abilitydefs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

func SpawnAbility(cooldown int) (ecs.AbilityEnum, ecs.AbilityDef) {
	abiFunc := func(self common.EntityId, targets []common.EntityId, targetPos utils.Vec2, world *ecs.World) error {
		if world.Animations.HasComponent(self) {
			am := world.AnimationManager

			nextState, err := am.GetState(self, world)
			if err != nil {
				return fmt.Errorf("error getting animation state for entity %d: %v", self, err)
			}

			err = am.SetQueuedStateIfNone(self, nextState, world)
			if err != nil {
				return fmt.Errorf("error setting queued animation state for entity %d: %v", self, err)
			}

			// TODO: Maybe decouple the animation here by adding it as a parameter to SpawnAbility
			err = am.SetState(self, ecs.Anim_Use, world)
			if err != nil {
				return fmt.Errorf("error setting animation state for entity %d: %v", self, err)
			}
		}

		if world.Spawners.HasComponent(self) {
			sm := world.SpawnerManager
			_, err := sm.Spawn(self, world)
			if err != nil {
				return fmt.Errorf("error spawning entity from spawner %d: %v", self, err)
			}
		}

		return nil
	}

	return ecs.Ability_Spawn, ecs.NewAbilityDef(
		abiFunc,
		cooldown,
		0,
		nil,
	)
}
