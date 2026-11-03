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
	target ecs.AbilityTarget,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) (bool, error) {
	if params == nil {
		return false, fmt.Errorf("dodge ability called with nil params")
	}

	dodgeParams, ok := params.(*DodgeParams)
	if !ok {
		return false, fmt.Errorf("dodge ability called with invalid params type: %T", params)
	}
	if dodgeParams == nil {
		return false, fmt.Errorf("dodge ability called with nil params")
	}

	anim, animOk := ecsContainer.Animations.Get(self)
	inv, invOk := ecsContainer.Invulnerabilities.Get(self)
	spr, sprOk := ecsContainer.Sprites.Get(self)
	if !animOk || !invOk || !sprOk {
		return false, fmt.Errorf("entity %d needs an animation, invulnerability and a sprite to dodge", self)
	}

	err := ecsContainer.SetAnimationState(self, ecs.Anim_Jump)
	if err != nil {
		return false, fmt.Errorf("error setting animation state of entity %d to jump: %v", self, err)
	}

	err = anim.SetQueuedStateIfNone(ecs.Anim_Idle)
	if err != nil {
		return false, fmt.Errorf("error setting queued animation state of entity %d to idle: %v", self, err)
	}

	inv.Set(math.MaxInt)

	err = spr.SetFlash(
		[]utils.RelativeColor{
			{R: 10, G: 10, B: 10, A: 1},
		},
		[]int{1000},
		math.MaxInt,
	)
	if err != nil {
		return false, fmt.Errorf("error setting sprite flash of entity %d: %v", self, err)
	}

	is, err := ecsContainer.GetCurrentTickInputsForEntity(self)
	if err != nil {
		return false, fmt.Errorf("error getting current tick inputs for entity %d: %v", self, err)
	}

	dir := utils.Vec2f{X: is.Analog1X, Y: is.Analog1Y}.Normalized()

	v, ok := ecsContainer.Velocities.Get(self)
	if !ok {
		return false, fmt.Errorf("entity %d has no velocity to dodge with", self)
	}
	v.AddForce(dir.Multiply(dodgeParams.Force))

	return true, nil
}

func DodgeAbilityPost(
	self common.EntityId,
	target ecs.AbilityTarget,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) (bool, error) {
	inv, invOk := ecsContainer.Invulnerabilities.Get(self)
	spr, sprOk := ecsContainer.Sprites.Get(self)
	if !invOk || !sprOk {
		return false, fmt.Errorf("entity %d needs invulnerability and a sprite to end its dodge", self)
	}

	inv.Set(0)
	spr.StopFlash()

	return true, nil
}
