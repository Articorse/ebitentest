package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

type DodgeParams struct {
	Force float64
}

func (DodgeParams) IsAbilityParams() {}

func DodgeAbility(
	self common.EntityId,
	params AbilityParams,
	ecsContainer *ECSContainer,
) error {
	if params == nil {
		return fmt.Errorf("dodge ability called with nil params")
	}

	dodgeParams, ok := params.(DodgeParams)
	if !ok {
		return fmt.Errorf("dodge ability called with invalid params type: %T", params)
	}

	am := ecsContainer.AnimationManager
	hpm := ecsContainer.HitpointsManager
	vm := ecsContainer.VelocityManager
	sm := ecsContainer.SpriteManager

	err := am.SetState(self, Anim_Jump, ecsContainer)
	if err != nil {
		return fmt.Errorf("error setting animation state of entity %d to jump: %v", self, err)
	}

	err = am.SetQueuedStateIfNone(self, Anim_Idle, ecsContainer)
	if err != nil {
		return fmt.Errorf("error setting queued animation state of entity %d to idle: %v", self, err)
	}

	err = hpm.SetInvul(self, math.MaxInt, ecsContainer)
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

	dir := utils.Vec2f{X: is.Analog1X, Y: is.Analog1Y}.Normalized()

	err = vm.AddForce(self, dir.Multiply(dodgeParams.Force), ecsContainer)
	if err != nil {
		return fmt.Errorf("error adding force to entity %d: %v", self, err)
	}

	return nil
}

func DodgeAbilityPost(
	self common.EntityId,
	params AbilityParams,
	ecsContainer *ECSContainer,
) error {
	hpm := ecsContainer.HitpointsManager
	sm := ecsContainer.SpriteManager

	err := hpm.SetInvul(self, 0, ecsContainer)
	if err != nil {
		return fmt.Errorf("error setting invulnerability of entity %d to 0: %v", self, err)
	}

	err = sm.StopFlash(self, ecsContainer)
	if err != nil {
		return fmt.Errorf("error clearing sprite flash of entity %d: %v", self, err)
	}

	return nil
}
