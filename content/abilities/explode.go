package abilities

import (
	"ebittest/content/timerfuncs"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
	"slices"
)

type ExplodeParams struct {
	Force           float64
	Radii           []float64
	DmgTiers        []int
	Animationframes []ecs.AnimationFrame
	SelfDestruct    bool
}

func (ExplodeParams) IsAbilityParams() {}

func (p *ExplodeParams) Copy() ecs.AbilityParams {
	if p == nil {
		return p
	}
	c := *p
	c.Radii = slices.Clone(p.Radii)
	c.DmgTiers = slices.Clone(p.DmgTiers)
	c.Animationframes = slices.Clone(p.Animationframes)
	return &c
}

func ExplodeAbility(
	self common.EntityId,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) error {
	if params == nil {
		return fmt.Errorf("explode ability params cannot be nil")
	}

	explodeParams, ok := params.(*ExplodeParams)
	if !ok {
		return fmt.Errorf("invalid explode ability params type: %T", params)
	}
	if explodeParams == nil {
		return fmt.Errorf("explode ability params cannot be nil")
	}

	var totalDuration uint64
	explosionE := ecsContainer.AddEmptyEntity()

	if !ecsContainer.Transforms.HasComponent(self) {
		return fmt.Errorf("entity %d does not have a transform component for explode ability", self)
	}

	for _, f := range explodeParams.Animationframes {
		totalDuration += f.DurationMs
	}

	exWorldPos, err := ecsContainer.WorldPos(self)
	if err != nil {
		return fmt.Errorf("error getting world position of entity %d for explode ability: %v", self, err)
	}

	exWorldRot, err := ecsContainer.WorldRotation(self)
	if err != nil {
		return fmt.Errorf("error getting world rotation of entity %d for explode ability: %v", self, err)
	}

	exWorldScale, err := ecsContainer.WorldScale(self)
	if err != nil {
		return fmt.Errorf("error getting world scale of entity %d for explode ability: %v", self, err)
	}

	var exSprLayer uint8

	if spr, ok := ecsContainer.Sprites.Get(self); ok {
		exSprLayer = spr.Layer()
	}

	hurtbox, ok := ecsContainer.HurtboxColliders.Get(self)
	if !ok {
		return fmt.Errorf("entity %d does not have a hurtbox collider component for explode ability", self)
	}

	dmgLayer := hurtbox.Layer()
	dmgMask := hurtbox.Mask()

	traComp := ecs.NewTransformComponent(exWorldPos, exWorldScale, exWorldRot)
	sprComp, err := ecs.NewSpriteComponent(common.AssetSheetExplosion, exSprLayer+1, false)
	if err != nil {
		return fmt.Errorf("error creating sprite component for explode ability: %v", err)
	}
	stateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	stateFrames[ecs.Anim_Idle] = explodeParams.Animationframes
	aniComp, err := ecs.NewAnimationComponent(common.AssetSheetExplosion, stateFrames)
	if err != nil {
		return fmt.Errorf("error creating animation component for explode ability: %v", err)
	}
	timerComp, err := ecs.NewTimerComponent(int(totalDuration), 1, timerfuncs.TimerFunc_Selfdestruct)
	if err != nil {
		return fmt.Errorf("error creating timer component for explode ability: %v", err)
	}
	hurtShapes := make([]shapes.Shape, len(explodeParams.Radii))
	for i := range explodeParams.Radii {
		hs, err := shapes.NewCircleShape(explodeParams.Radii[i], utils.Vec2f{})
		if err != nil {
			return fmt.Errorf("error creating hurtbox shape for explode ability: %v", err)
		}
		hurtShapes[i] = hs
	}

	hurtComp := ecs.NewHurtboxColliderComponent(dmgLayer, dmgMask, hurtShapes...)
	cdComp := ecs.NewContactDamageComponent(self, explodeParams.Force, false, true, explodeParams.DmgTiers...)

	ecsContainer.AddComponent(explosionE, traComp)
	ecsContainer.AddComponent(explosionE, sprComp)
	ecsContainer.AddComponent(explosionE, aniComp)
	ecsContainer.AddComponent(explosionE, timerComp)
	ecsContainer.AddComponent(explosionE, cdComp)
	ecsContainer.AddComponent(explosionE, hurtComp)

	return nil
}
