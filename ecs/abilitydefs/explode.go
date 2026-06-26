package abilitydefs

import (
	"ebittest/assetmanager"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/timerfuncs"
	"ebittest/utils"
	"fmt"
)

func ExplodeAbility(
	force float64,
	radii []float64,
	dmgTiers []int,
	animationframes []ecs.AnimationFrame,
	selfDestruct bool,
	assetManager *assetmanager.AssetManager,
) (ecs.AbilityEnum, ecs.AbilityDef, error) {
	if len(radii) != len(dmgTiers) {
		return ecs.Ability_None, ecs.AbilityDef{}, fmt.Errorf("explodeAbility created with mismatched radii and damage tiers lengths: %d vs %d", len(radii), len(dmgTiers))
	}

	abiFunc := func(self common.EntityId, targets []common.EntityId, targetPos utils.Vec2, ecsContainer *ecs.ECSContainer) error {
		var totalDuration uint64
		explosionE := ecsContainer.AddEmptyEntity()

		if !ecsContainer.Transforms.HasComponent(self) {
			return fmt.Errorf("entity %d does not have a transform component for explode ability", self)
		}

		for _, f := range animationframes {
			totalDuration += f.DurationMs
		}

		exWorldPos, err := ecsContainer.TransformManager.GetWorldPos(self, ecsContainer)
		if err != nil {
			return fmt.Errorf("error getting world position of entity %d for explode ability: %v", self, err)
		}

		exWorldRot, err := ecsContainer.TransformManager.GetWorldRotation(self, ecsContainer)
		if err != nil {
			return fmt.Errorf("error getting world rotation of entity %d for explode ability: %v", self, err)
		}

		exWorldScale, err := ecsContainer.TransformManager.GetWorldScale(self, ecsContainer)
		if err != nil {
			return fmt.Errorf("error getting world scale of entity %d for explode ability: %v", self, err)
		}

		var exSprLayer uint8

		if ecsContainer.Sprites.HasComponent(self) {
			exSprLayer, err = ecsContainer.SpriteManager.GetLayer(self, ecsContainer)
			if err != nil {
				return fmt.Errorf("error getting sprite layer of entity %d for explode ability: %v", self, err)
			}
		}

		if !ecsContainer.HurtboxColliders.HasComponent(self) {
			return fmt.Errorf("entity %d does not have a hurtbox collider component for explode ability", self)
		}

		dmgLayer, err := ecsContainer.HurtboxColliderManager.GetLayer(self, ecsContainer)
		if err != nil {
			return fmt.Errorf("error getting hurtbox collider layer of entity %d for explode ability: %v", self, err)
		}

		dmgMask, err := ecsContainer.HurtboxColliderManager.GetMask(self, ecsContainer)
		if err != nil {
			return fmt.Errorf("error getting hurtbox collider mask of entity %d for explode ability: %v", self, err)
		}

		traComp := ecs.NewTransformComponent(exWorldPos, exWorldScale, exWorldRot)
		sprComp, err := ecs.NewSpriteComponent("", exSprLayer+1, false, assetManager)
		if err != nil {
			return fmt.Errorf("error creating sprite component for explode ability: %v", err)
		}
		stateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
		stateFrames[ecs.Anim_Idle] = animationframes
		aniComp, err := ecs.NewAnimationComponent(assetmanager.AssetSheetExplosion, stateFrames)
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

		ecsContainer.AddComponent(explosionE, traComp)
		ecsContainer.AddComponent(explosionE, sprComp)
		ecsContainer.AddComponent(explosionE, aniComp)
		ecsContainer.AddComponent(explosionE, timerComp)
		ecsContainer.AddComponent(explosionE, cdComp)
		ecsContainer.AddComponent(explosionE, hurtComp)

		return nil
	}

	return ecs.Ability_Explode, ecs.NewAbilityDef(
		abiFunc,
		0,
		0,
		nil,
	), nil
}
