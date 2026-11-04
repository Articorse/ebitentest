package abilities

import (
	"ebittest/content/timerfuncs"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
	"math"
	"slices"
)

type SlashParams struct {
	Image common.ImageAssetTag
	// The area the slash hits while it points right, around the center of its image. Convex, since that's all the
	// collision checks handle.
	Hull       []utils.Vec2f
	ReachPx    float64
	Damage     int
	Mask       ecs.LayerMask
	DurationMs int
}

func (SlashParams) IsAbilityParams() {}

func (p *SlashParams) Copy() ecs.AbilityParams {
	if p == nil {
		return p
	}
	c := *p
	c.Hull = slices.Clone(p.Hull)
	return &c
}

func (p *SlashParams) UsedImages() map[common.ImageAssetTag][]int {
	return map[common.ImageAssetTag][]int{p.Image: {0}}
}

// Leaves a slash toward the aim, at the reach from whoever holds the item, which hurts what it covers on the tick it
// appears and stays in place until it fades
func SlashAbility(
	self common.EntityId,
	target ecs.AbilityTarget,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) (bool, error) {
	p, ok := params.(*SlashParams)
	if !ok || p == nil {
		return false, fmt.Errorf("slash ability called with invalid params: %T", params)
	}

	slasher := self
	if holder := ecsContainer.Parent(self); holder != -1 {
		slasher = holder
	}
	slasherPos, err := ecsContainer.WorldPos(slasher)
	if err != nil {
		return false, fmt.Errorf("error getting world position of entity %d for slashing: %v", slasher, err)
	}

	if !target.HasPos || target.Pos == slasherPos {
		return false, nil
	}
	toAim := target.Pos.Subtract(slasherPos)
	angle := math.Atan2(toAim.Y, toAim.X)

	shape, err := shapes.NewPolygonShape(slices.Clone(p.Hull), utils.Vec2f{})
	if err != nil || shape == nil {
		return false, fmt.Errorf("error creating the hull of a slash from %v: %v", p.Hull, err)
	}

	var layer uint8
	if spr, ok := ecsContainer.Sprites.Get(self); ok {
		layer = spr.Layer() + 1
	}
	spr, err := ecs.NewSpriteComponent(p.Image, layer, true)
	if err != nil {
		return false, fmt.Errorf("error creating the sprite of a slash: %v", err)
	}
	timer, err := ecs.NewTimerComponent(p.DurationMs, 1, timerfuncs.TimerFunc_Selfdestruct)
	if err != nil {
		return false, fmt.Errorf("error creating the timer of a slash: %v", err)
	}

	if _, err := ecsContainer.AddNewEntity(
		ecs.NewTransformComponent(slasherPos.Add(toAim.Normalized().Multiply(p.ReachPx)), 1, angle),
		spr,
		timer,
		ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, p.Mask, shape),
		ecs.NewContactDamageComponent(slasher, 0, p.Damage).WithActiveMs(data.TickMs),
	); err != nil {
		return false, fmt.Errorf("error spawning a slash: %v", err)
	}

	return true, nil
}
