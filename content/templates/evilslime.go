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

// Rushes at the closest hostile it sees and hurts it on contact. Its body is the player's, so it's hurt the same way,
// and it sees half as far.
func SpawnEvilSlime(ecsContainer *ecs.ECSContainer, pos utils.Vec2f) (common.EntityId, error) {
	spr, err := ecs.NewSpriteComponent(common.AssetSheetEvilSlime, 20, false)
	if err != nil {
		return -1, err
	}
	// Waits squatting, then hops up and back down
	const hopFrameMs = 300 / 7
	ani, err := ecs.NewAnimationComponent(common.AssetSheetEvilSlime, map[ecs.AnimationState][]ecs.AnimationFrame{
		ecs.Anim_Idle: {
			{FrameIdx: 0, DurationMs: 2000},
			{FrameIdx: 1, DurationMs: hopFrameMs},
			{FrameIdx: 2, DurationMs: hopFrameMs},
			{FrameIdx: 3, DurationMs: hopFrameMs},
			{FrameIdx: 4, DurationMs: hopFrameMs},
			{FrameIdx: 3, DurationMs: hopFrameMs},
			{FrameIdx: 2, DurationMs: hopFrameMs},
			{FrameIdx: 1, DurationMs: hopFrameMs},
		},
	})
	if err != nil {
		return -1, err
	}

	physicsShape, err := shapes.NewCircleShape(5, utils.Vec2f{})
	if err != nil {
		return -1, err
	}
	hitboxShape, err := shapes.NewCircleShape(5, utils.Vec2f{})
	if err != nil {
		return -1, err
	}
	hurtboxShape, err := shapes.NewCircleShape(5, utils.Vec2f{})
	if err != nil {
		return -1, err
	}

	slime, err := ecsContainer.AddNewEntity(
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(pos, 1, 0),
		ecs.NewVelocityComponentWithParams(utils.Vec2f{}, data.DefaultAcceleration*0.5, data.DefaultDrag),
		spr,
		ani,
		ecs.NewPhysicsColliderComponent(
			ecs.Layer_Enemy,
			ecs.Layer_Player|
				ecs.Layer_Enemy|
				ecs.Layer_Terrain|
				ecs.Layer_Water|
				ecs.Layer_Feature|
				ecs.Layer_Platform,
			ecs.Collider_Mob,
			physicsShape,
		),
		ecs.NewHitboxColliderComponent(ecs.Layer_Enemy, ecs.Layer_FriendlyProjectile, hitboxShape),
		ecs.NewHurtboxColliderComponent(ecs.Layer_Enemy, ecs.Layer_Player, hurtboxShape),
		ecs.NewContactDamageComponent(-1, 5, 5).WithRehitIntervalMs(1000),
		ecs.NewBodyComponent(data.Body_Humanoid),
		ecs.NewFactionComponent(data.Faction_Monsters),
		ecs.NewInputComponent(nil, inputsources.InputType_Rusher, &inputsources.InputRusherParams{Target: -1}, ecs.Facing_None),
		ecs.NewVisionComponent(data.PlayerVisionRangeTiles/2),
		ecs.NewPersistentComponent(),
	)
	if err != nil {
		return -1, fmt.Errorf("error spawning evil slime: %w", err)
	}
	return slime, nil
}
