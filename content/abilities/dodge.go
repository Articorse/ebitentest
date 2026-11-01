package abilities

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

type DodgeParams struct {
	Force float64
}

func (DodgeParams) IsAbilityParams() {}

func (p *DodgeParams) Copy() ecs.AbilityParams {
	if p == nil {
		return p
	}
	c := *p
	return &c
}

func DodgeAbility(
	self common.EntityId,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) error {
	if params == nil {
		return fmt.Errorf("dodge ability called with nil params")
	}

	dodgeParams, ok := params.(*DodgeParams)
	if !ok {
		return fmt.Errorf("dodge ability called with invalid params type: %T", params)
	}
	if dodgeParams == nil {
		return fmt.Errorf("dodge ability called with nil params")
	}

	anim, animOk := ecsContainer.Animations.Get(self)
	hp, hpOk := ecsContainer.Hitpoints.Get(self)
	spr, sprOk := ecsContainer.Sprites.Get(self)
	if !animOk || !hpOk || !sprOk {
		return fmt.Errorf("entity %d needs an animation, hitpoints and a sprite to dodge", self)
	}

	err := ecsContainer.SetAnimationState(self, ecs.Anim_Jump)
	if err != nil {
		return fmt.Errorf("error setting animation state of entity %d to jump: %v", self, err)
	}

	err = anim.SetQueuedStateIfNone(ecs.Anim_Idle)
	if err != nil {
		return fmt.Errorf("error setting queued animation state of entity %d to idle: %v", self, err)
	}

	hp.SetInvul(math.MaxInt)

	err = spr.SetFlash(
		[]utils.RelativeColor{
			{R: 10, G: 10, B: 10, A: 1},
		},
		[]int{1000},
		math.MaxInt,
	)
	if err != nil {
		return fmt.Errorf("error setting sprite flash of entity %d: %v", self, err)
	}

	is, err := ecsContainer.GetCurrentTickInputsForEntity(self)
	if err != nil {
		return fmt.Errorf("error getting current tick inputs for entity %d: %v", self, err)
	}

	dir := utils.Vec2f{X: is.Analog1X, Y: is.Analog1Y}.Normalized()

	v, ok := ecsContainer.Velocities.Get(self)
	if !ok {
		return fmt.Errorf("entity %d has no velocity to dodge with", self)
	}
	v.AddForce(dir.Multiply(dodgeParams.Force))

	return nil
}

func DodgeAbilityPost(
	self common.EntityId,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) error {
	hp, hpOk := ecsContainer.Hitpoints.Get(self)
	spr, sprOk := ecsContainer.Sprites.Get(self)
	if !hpOk || !sprOk {
		return fmt.Errorf("entity %d needs hitpoints and a sprite to end its dodge", self)
	}

	hp.SetInvul(0)
	spr.StopFlash()

	return nil
}
