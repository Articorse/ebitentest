package templates

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

// Around the giant slime's body in its sheet, from the middle of a frame, smoothed rather than following its pixels.
// Its top is between the tallest and the most squashed frame's.
var giantSlimeOutline = []utils.Vec2f{
	{X: -12, Y: -24}, {X: 12, Y: -24}, {X: 28, Y: -12}, {X: 36, Y: 4}, {X: 40, Y: 16},
	{X: 32, Y: 31}, {X: -32, Y: 31}, {X: -40, Y: 16}, {X: -36, Y: 4}, {X: -28, Y: -12},
}

// Shrunk by 2 pixels all round, so that what it hurts on contact has to have reached a little into it
var giantSlimeHurtOutline = []utils.Vec2f{
	{X: -11, Y: -22}, {X: 11, Y: -22}, {X: 26, Y: -11}, {X: 34, Y: 4}, {X: 38, Y: 16},
	{X: 30, Y: 29}, {X: -30, Y: 29}, {X: -38, Y: 16}, {X: -34, Y: 4}, {X: -26, Y: -11},
}

// Half the outline round the middle of its body, as the smaller slimes' bodies are about twice as wide as what they
// bump into things with
var giantSlimeFootprint = []utils.Vec2f{
	{X: -6, Y: -11}, {X: 6, Y: -11}, {X: 14, Y: -5}, {X: 18, Y: 3}, {X: 20, Y: 9},
	{X: 16, Y: 17}, {X: -16, Y: 17}, {X: -20, Y: 9}, {X: -18, Y: 3}, {X: -14, Y: -5},
}

func GiantSlimePhysicsShape() (*shapes.PolygonShape, error) {
	return shapes.NewPolygonShape(giantSlimeFootprint, utils.Vec2f{})
}

// An evil slime four times as big, with four times its health and a third of its speed. Animated as the slimes of the
// colony are, whose sheets it has the frames of.
func SpawnGiantSlime(ecsContainer *ecs.ECSContainer, pos utils.Vec2f) (common.EntityId, error) {
	spr, err := ecs.NewSpriteComponent(common.AssetSheetGiantSlime, 20, false)
	if err != nil {
		return -1, err
	}
	anim, err := ecs.NewAnimationComponent(common.AssetSheetGiantSlime, colonySlimeFrames())
	if err != nil {
		return -1, err
	}
	physicsShape, err := GiantSlimePhysicsShape()
	if err != nil {
		return -1, err
	}
	hitboxShape, err := shapes.NewPolygonShape(giantSlimeOutline, utils.Vec2f{})
	if err != nil {
		return -1, err
	}
	hurtboxShape, err := shapes.NewPolygonShape(giantSlimeHurtOutline, utils.Vec2f{})
	if err != nil {
		return -1, err
	}

	slime, err := ecsContainer.AddNewEntity(
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(pos, 1, 0),
		ecs.NewVelocityComponentWithParams(utils.Vec2f{}, data.DefaultAcceleration*0.5/3, data.DefaultDrag),
		spr,
		anim,
		ecs.NewPhysicsColliderComponent(
			ecs.Layer_Enemy,
			ecs.Layer_Player|
				ecs.Layer_Enemy|
				ecs.Layer_Wall|
				ecs.Layer_Water|
				ecs.Layer_Feature|
				ecs.Layer_Platform,
			ecs.Collider_Mob,
			physicsShape,
		),
		ecs.NewHitboxColliderComponent(ecs.Layer_Enemy, ecs.Layer_FriendlyProjectile, hitboxShape),
		ecs.NewHurtboxColliderComponent(ecs.Layer_Enemy, ecs.Layer_Player, hurtboxShape),
		ecs.NewContactDamageComponent(-1, 5, 5).WithRehitIntervalMs(1000),
		ecs.NewBodyComponent(data.Body_Giant),
		ecs.NewFactionComponent(data.Faction_Monsters),
		ecs.NewInputComponent(nil, inputsources.InputType_Rusher, &inputsources.InputRusherParams{Target: -1}, ecs.Facing_None),
		ecs.NewVisionComponent(data.PlayerVisionRangeTiles/2, data.EnemyVisionEveryTicks),
		ecs.NewPersistentComponent(),
	)
	if err != nil {
		return -1, fmt.Errorf("error spawning giant slime: %w", err)
	}
	return slime, nil
}
