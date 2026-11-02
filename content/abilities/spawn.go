package abilities

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
)

func SpawnAbility(
	self common.EntityId,
	target ecs.AbilityTarget,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) (bool, error) {
	if anim, ok := ecsContainer.Animations.Get(self); ok {
		err := anim.SetQueuedStateIfNone(anim.State())
		if err != nil {
			return false, fmt.Errorf("error setting queued animation state for entity %d: %v", self, err)
		}

		// TODO: Maybe decouple the animation here by adding it as a parameter to SpawnAbility
		err = ecsContainer.SetAnimationState(self, ecs.Anim_Use)
		if err != nil {
			return false, fmt.Errorf("error setting animation state for entity %d: %v", self, err)
		}
	}

	if ecsContainer.Spawners.HasComponent(self) {
		_, err := ecsContainer.Spawn(self)
		if err != nil {
			return false, fmt.Errorf("error spawning entity from spawner %d: %v", self, err)
		}
	}

	return true, nil
}
