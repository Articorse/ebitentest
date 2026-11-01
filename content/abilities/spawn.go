package abilities

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
)

func SpawnAbility(
	self common.EntityId,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) error {
	if ecsContainer.Animations.HasComponent(self) {
		am := ecsContainer.AnimationManager

		nextState, err := am.GetState(self, ecsContainer)
		if err != nil {
			return fmt.Errorf("error getting animation state for entity %d: %v", self, err)
		}

		err = am.SetQueuedStateIfNone(self, nextState, ecsContainer)
		if err != nil {
			return fmt.Errorf("error setting queued animation state for entity %d: %v", self, err)
		}

		// TODO: Maybe decouple the animation here by adding it as a parameter to SpawnAbility
		err = am.SetState(self, ecs.Anim_Use, ecsContainer)
		if err != nil {
			return fmt.Errorf("error setting animation state for entity %d: %v", self, err)
		}
	}

	if ecsContainer.Spawners.HasComponent(self) {
		sm := ecsContainer.SpawnerManager
		_, err := sm.Spawn(self, ecsContainer)
		if err != nil {
			return fmt.Errorf("error spawning entity from spawner %d: %v", self, err)
		}
	}

	return nil
}
