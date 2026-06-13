package abilitydefs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

const (
	Dodge_Cooldown = 1000
	Dodge_Duration = 200
	Dodge_Force    = 10
)

func DodgeAbility() (ecs.AbilityEnum, ecs.AbilityDef) {
	abiFunc := func(self common.EntityId, targets []common.EntityId, world *ecs.World) error {
		am := ecs.AnimationManager{}
		hpm := ecs.HitpointsManager{}
		vm := ecs.VelocityManager{}
		sm := ecs.SpriteManager{}

		err := am.SetState(self, ecs.Anim_Jump, world)
		if err != nil {
			return fmt.Errorf("error setting animation state of entity %d to jump: %v", self, err)
		}

		err = am.SetQueuedStateIfNone(self, ecs.Anim_Idle, world)
		if err != nil {
			return fmt.Errorf("error setting queued animation state of entity %d to idle: %v", self, err)
		}

		err = hpm.SetInvul(self, math.MaxInt, world)
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
			world,
		)
		if err != nil {
			return fmt.Errorf("error setting sprite flash of entity %d: %v", self, err)
		}

		is, err := world.GetCurrentTickInputsForEntity(self)
		if err != nil {
			return fmt.Errorf("error getting current tick inputs for entity %d: %v", self, err)
		}

		var dir utils.Vec2
		if is.Analog1X > 0 {
			dir.X = 1
		}
		if is.Analog1X < 0 {
			dir.X = -1
		}
		if is.Analog1Y > 0 {
			dir.Y = 1
		}
		if is.Analog1Y < 0 {
			dir.Y = -1
		}

		dir = dir.Normalized()

		err = vm.AddForce(self, dir.Multiply(Dodge_Force), world)
		if err != nil {
			return fmt.Errorf("error adding force to entity %d: %v", self, err)
		}

		return nil
	}

	abiPostFunc := func(self common.EntityId, targets []common.EntityId, world *ecs.World) error {
		hpm := ecs.HitpointsManager{}
		sm := ecs.SpriteManager{}

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
		Dodge_Cooldown,
		Dodge_Duration,
		abiPostFunc,
	)
}
