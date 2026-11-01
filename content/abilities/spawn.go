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
	if anim, ok := ecsContainer.Animations.Get(self); ok {
		err := anim.SetQueuedStateIfNone(anim.State())
		if err != nil {
			return fmt.Errorf("error setting queued animation state for entity %d: %v", self, err)
		}

		// TODO: Maybe decouple the animation here by adding it as a parameter to SpawnAbility
		err = ecsContainer.SetAnimationState(self, ecs.Anim_Use)
		if err != nil {
			return fmt.Errorf("error setting animation state for entity %d: %v", self, err)
		}
	}

	if ecsContainer.Spawners.HasComponent(self) {
		_, err := ecsContainer.Spawn(self)
		if err != nil {
			return fmt.Errorf("error spawning entity from spawner %d: %v", self, err)
		}
	}

	return nil
}
