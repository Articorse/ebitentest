package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

type ExplodeParams struct {
	Force           float64
	Radii           []float64
	DmgTiers        []int
	Animationframes []AnimationFrame
	SelfDestruct    bool
}

func (ExplodeParams) IsAbilityParams() {}

func ExplodeAbility(
	self common.EntityId,
	params AbilityParams,
	ecsContainer *ECSContainer,
) error {
	if params == nil {
		return fmt.Errorf("explode ability params cannot be nil")
	}

	explodeParams, ok := params.(ExplodeParams)
	if !ok {
		return fmt.Errorf("invalid explode ability params type")
	}

	var totalDuration uint64
	explosionE := ecsContainer.AddEmptyEntity()

	if !ecsContainer.Transforms.HasComponent(self) {
		return fmt.Errorf("entity %d does not have a transform component for explode ability", self)
	}

	for _, f := range explodeParams.Animationframes {
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

	traComp := NewTransformComponent(exWorldPos, exWorldScale, exWorldRot)
	sprComp, err := NewSpriteComponent(common.AssetSheetExplosion, exSprLayer+1, false)
	if err != nil {
		return fmt.Errorf("error creating sprite component for explode ability: %v", err)
	}
	stateFrames := make(map[AnimationState][]AnimationFrame)
	stateFrames[Anim_Idle] = explodeParams.Animationframes
	aniComp, err := NewAnimationComponent(common.AssetSheetExplosion, stateFrames)
	if err != nil {
		return fmt.Errorf("error creating animation component for explode ability: %v", err)
	}
	timerComp, err := NewTimerComponent(int(totalDuration), 1, TimerFunc_Selfdestruct)
	if err != nil {
		return fmt.Errorf("error creating timer component for explode ability: %v", err)
	}
	hurtShapes := make([]shapes.Shape, len(explodeParams.Radii))
	for i := range explodeParams.Radii {
		hs, err := shapes.NewCircleShape(explodeParams.Radii[i], utils.Vec2{})
		if err != nil {
			return fmt.Errorf("error creating hurtbox shape for explode ability: %v", err)
		}
		hurtShapes[i] = hs
	}

	hurtComp := NewHurtboxColliderComponent(dmgLayer, dmgMask, hurtShapes...)
	cdComp := NewContactDamageComponent(self, explodeParams.Force, false, true, explodeParams.DmgTiers...)

	ecsContainer.AddComponent(explosionE, traComp)
	ecsContainer.AddComponent(explosionE, sprComp)
	ecsContainer.AddComponent(explosionE, aniComp)
	ecsContainer.AddComponent(explosionE, timerComp)
	ecsContainer.AddComponent(explosionE, cdComp)
	ecsContainer.AddComponent(explosionE, hurtComp)

	return nil
}
