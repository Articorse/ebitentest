package abilitydefs

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/timerfuncs"
	"ebittest/utils"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

func ExplodeAbility(
	force float64,
	radii []float64,
	dmgTiers []int,
	spriteSheet *ebiten.Image,
	frameSize utils.Vec2,
	animationframes []ecs.AnimationFrame,
	selfDestruct bool,
	ecs *ecs.ECS,
) (ecs.AbilityEnum, ecs.AbilityDef, error) {
	if len(radii) != len(dmgTiers) {
		return ecs.Ability_None, ecs.AbilityDef{}, fmt.Errorf("explodeAbility created with mismatched radii and damage tiers lengths: %d vs %d", len(radii), len(dmgTiers))
	}

	abiFunc := func(self common.EntityId, targets []common.EntityId, targetPos utils.Vec2, ecs *ecs.ECS) error {
		var totalDuration uint64
		explosionE := ecs.AddEmptyEntity()

		if !ecs.Transforms.HasComponent(self) {
			return fmt.Errorf("entity %d does not have a transform component for explode ability", self)
		}

		for _, f := range animationframes {
			totalDuration += f.DurationMs
		}

		exWorldPos, err := ecs.TransformManager.GetWorldPos(self, ecs)
		if err != nil {
			return fmt.Errorf("error getting ecs position of entity %d for explode ability: %v", self, err)
		}

		exWorldRot, err := ecs.TransformManager.GetWorldRotation(self, ecs)
		if err != nil {
			return fmt.Errorf("error getting ecs rotation of entity %d for explode ability: %v", self, err)
		}

		exWorldScale, err := ecs.TransformManager.GetWorldScale(self, ecs)
		if err != nil {
			return fmt.Errorf("error getting ecs scale of entity %d for explode ability: %v", self, err)
		}

		var exSprLayer uint8

		if ecs.Sprites.HasComponent(self) {
			exSprLayer, err = ecs.SpriteManager.GetLayer(self, ecs)
			if err != nil {
				return fmt.Errorf("error getting sprite layer of entity %d for explode ability: %v", self, err)
			}
		}

		if !ecs.HurtboxColliders.HasComponent(self) {
			return fmt.Errorf("entity %d does not have a hurtbox collider component for explode ability", self)
		}

		dmgLayer, err := ecs.HurtboxColliderManager.GetLayer(self, ecs)
		if err != nil {
			return fmt.Errorf("error getting hurtbox collider layer of entity %d for explode ability: %v", self, err)
		}

		dmgMask, err := ecs.HurtboxColliderManager.GetMask(self, ecs)
		if err != nil {
			return fmt.Errorf("error getting hurtbox collider mask of entity %d for explode ability: %v", self, err)
		}

		traComp := ecs.NewTransformComponent(exWorldPos, exWorldScale, exWorldRot)
		sprComp, err := ecs.NewSpriteComponent("", exSprLayer+1, false)
		if err != nil {
			return fmt.Errorf("error creating sprite component for explode ability: %v", err)
		}
		stateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
		stateFrames[ecs.Anim_Idle] = animationframes
		aniComp, err := ecs.NewAnimationComponentWithSheet(spriteSheet, frameSize, stateFrames)
		if err != nil {
			return fmt.Errorf("error creating animation component for explode ability: %v", err)
		}
		timerComp, err := ecs.NewTimerComponent(int(totalDuration), 1, timerfuncs.Selfdestruct)
		if err != nil {
			return fmt.Errorf("error creating timer component for explode ability: %v", err)
		}
		hurtShapes := make([]shapes.Shape, len(radii))
		for i := range radii {
			hs, err := shapes.NewCircleShape(radii[i], utils.Vec2{})
			if err != nil {
				return fmt.Errorf("error creating hurtbox shape for explode ability: %v", err)
			}
			hurtShapes[i] = hs
		}

		hurtComp := ecs.NewHurtboxColliderComponent(dmgLayer, dmgMask, hurtShapes...)
		cdComp := ecs.NewContactDamageComponent(self, force, false, true, dmgTiers...)

		ecs.AddComponent(explosionE, traComp)
		ecs.AddComponent(explosionE, sprComp)
		ecs.AddComponent(explosionE, aniComp)
		ecs.AddComponent(explosionE, timerComp)
		ecs.AddComponent(explosionE, cdComp)
		ecs.AddComponent(explosionE, hurtComp)

		return nil
	}

	return ecs.Ability_Explode, ecs.NewAbilityDef(
		abiFunc,
		0,
		0,
		nil,
	), nil
}
