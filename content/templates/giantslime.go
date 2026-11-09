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
// It's the smallest frame's, so that it never reaches past the body as drawn.
var giantSlimeOutline = []utils.Vec2f{
	{X: -12, Y: -16}, {X: 12, Y: -16}, {X: 30, Y: 2}, {X: 40, Y: 16}, {X: 40, Y: 24},
	{X: 32, Y: 31}, {X: -32, Y: 31}, {X: -40, Y: 24}, {X: -40, Y: 16}, {X: -30, Y: 2},
}

// Shrunk by 4 pixels all round, so that what it hurts on contact has to have reached well into it
var giantSlimeHurtOutline = []utils.Vec2f{
	{X: -10, Y: -12}, {X: 10, Y: -12}, {X: 26, Y: 5}, {X: 36, Y: 17}, {X: 36, Y: 22},
	{X: 30, Y: 27}, {X: -30, Y: 27}, {X: -36, Y: 22}, {X: -36, Y: 17}, {X: -26, Y: 5},
}

// A pixel inside the hurt outline, so that it can't push anything it doesn't hurt. It takes up 5 tiles across and 4
// down, 1 above the tile it's in the middle of and 2 below.
var giantSlimeFootprint = []utils.Vec2f{
	{X: -9, Y: -11}, {X: 9, Y: -11}, {X: 25, Y: 6}, {X: 35, Y: 17}, {X: 35, Y: 21},
	{X: 29, Y: 26}, {X: -29, Y: 26}, {X: -35, Y: 21}, {X: -35, Y: 17}, {X: -25, Y: 6},
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
