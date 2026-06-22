package abilitydefs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

func DodgeAbility(cooldownMs int, durationMs int, force float64) (ecs.AbilityEnum, ecs.AbilityDef) {
	abiFunc := func(self common.EntityId, targets []common.EntityId, targetPos utils.Vec2, ecsContainer *ecs.ECSContainer) error {
		am := ecsContainer.AnimationManager
		hpm := ecsContainer.HitpointsManager
		vm := ecsContainer.VelocityManager
		sm := ecsContainer.SpriteManager

		err := am.SetState(self, ecs.Anim_Jump, ecsContainer)
		if err != nil {
			return fmt.Errorf("error setting animation state of entity %d to jump: %v", self, err)
		}

		err = am.SetQueuedStateIfNone(self, ecs.Anim_Idle, ecsContainer)
		if err != nil {
			return fmt.Errorf("error setting queued animation state of entity %d to idle: %v", self, err)
		}

		err = hpm.SetInvul(self, durationMs, ecsContainer)
		if err != nil {
			return fmt.Errorf("error setting invulnerability of entity %d: %v", self, err)
		}

		err = sm.SetSpriteFlash(
			self,
			[]utils.RelativeColor{
				{R: 10, G: 10, B: 10, A: 1},
			},
			[]int{1000},
			math.MaxInt,
			ecsContainer,
		)
		if err != nil {
			return fmt.Errorf("error setting sprite flash of entity %d: %v", self, err)
		}

		is, err := ecsContainer.GetCurrentTickInputsForEntity(self)
		if err != nil {
			return fmt.Errorf("error getting current tick inputs for entity %d: %v", self, err)
		}

		dir := utils.Vec2{X: is.Analog1X, Y: is.Analog1Y}.Normalized()

		err = vm.AddForce(self, dir.Multiply(force), ecsContainer)
		if err != nil {
			return fmt.Errorf("error adding force to entity %d: %v", self, err)
		}

		return nil
	}

	abiPostFunc := func(self common.EntityId, targets []common.EntityId, targetPos utils.Vec2, world *ecs.ECSContainer) error {
		hpm := world.HitpointsManager
		sm := world.SpriteManager

		err := hpm.SetInvul(self, 0, world)
		if err != nil {
			return fmt.Errorf("error setting invulnerability of entity %d to 0: %v", self, err)
		}

		err = sm.StopFlash(self, world)
		if err != nil {
			return fmt.Errorf("error clearing sprite flash of entity %d: %v", self, err)
		}

		return nil
	}

	return ecs.Ability_Dodge, ecs.NewAbilityDef(
		abiFunc,
		cooldownMs,
		durationMs,
		abiPostFunc,
	)
}
