package templates

import (
	"ebittest/content/abilities"
	"ebittest/content/timerfuncs"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

func SpawnPlayer(ecsContainer *ecs.ECSContainer) (common.EntityId, error) {
	// Player
	pParComp := ecs.NewHierarchyComponent()
	pTraComp := ecs.NewTransformComponent(utils.Vec2f{X: 100, Y: 100}, 1, 0)
	pVelComp := ecs.NewDefaultVelocityComponent()
	pSprComp, err := ecs.NewSpriteComponent(common.AssetSheetSlime, 20, false)
	if err != nil {
		return -1, err
	}
	pAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	pAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 100},
		{FrameIdx: 1, DurationMs: 100},
		{FrameIdx: 2, DurationMs: 100},
		{FrameIdx: 1, DurationMs: 100},
	}
	pAniStateFrames[ecs.Anim_Jump] = []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 50},
		{FrameIdx: 1, DurationMs: 50},
		{FrameIdx: 2, DurationMs: 50},
		{FrameIdx: 1, DurationMs: 50},
	}
	pAniComp, err := ecs.NewAnimationComponent(common.AssetSheetSlime, pAniStateFrames)
	if err != nil {
		return -1, err
	}
	pPhyColShape, err := shapes.NewCircleShape(5, utils.Vec2f{X: 0, Y: 0})
	if err != nil {
		return -1, err
	}
	pPhyColComp := ecs.NewPhysicsColliderComponent(
		ecs.Layer_Player,
		ecs.Layer_Enemy|
			ecs.Layer_EnemyProjectile|
			ecs.Layer_Terrain|
			ecs.Layer_Platform,
		ecs.Collider_Mob,
		pPhyColShape,
	)
	pHpComp := ecs.NewHitpointsComponent(20, 1000)
	pHitboxShape, err := shapes.NewCircleShape(5, utils.Vec2f{X: 0, Y: 0})
	if err != nil {
		return -1, err
	}
	pHitboxComp := ecs.NewHitboxColliderComponent(
		ecs.Layer_Player,
		ecs.Layer_Enemy|
			ecs.Layer_EnemyProjectile|
			ecs.Layer_Terrain|
			ecs.Layer_Platform,
		pHitboxShape,
	)
	pAbis := [16]ecs.EntityAbility{}
	pAbis[0] = ecs.EntityAbility{
		Def: ecs.NewAbilityDef(
			abilities.Ability_Dodge,
			abilities.Ability_Dodge_Post,
			1000,
			200,
		),
		Status: ecs.AbilityStatus{State: ecs.AbiAct_Ready},
		Params: &abilities.DodgeParams{
			Force: 10,
		},
	}
	pAbiComp := ecs.NewAbilitiesComponent(pAbis)
	pCLComp := ecs.NewChunkLoaderComponent(3)
	player, err := ecsContainer.AddNewEntity(
		pParComp,
		pTraComp,
		pVelComp,
		pSprComp,
		pAniComp,
		pPhyColComp,
		pHpComp,
		pHitboxComp,
		pAbiComp,
		pCLComp,
		ecs.NewPersistentComponent(),
	)
	if err != nil {
		return -1, fmt.Errorf("error spawning player: %w", err)
	}

	// // Gun
	// gunParComp := ecs.NewParentComponent()
	// gunTraComp := ecs.NewTransformComponent(utils.Vec2{X: 100, Y: 100}, 1, 0)
	// gunVelComp := ecs.NewDefaultVelocityComponent()
	// gunSprComp, err := ecs.NewSpriteComponent("assets/sprites/gun.png", 20, true)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// gunAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	// gunAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
	// 	{FrameIdx: 0, DurationMs: 1000},
	// }
	// gunAniStateFrames[ecs.Anim_Use] = []ecs.AnimationFrame{
	// 	{FrameIdx: 1, DurationMs: 100},
	// }
	// gunAniComp, err := ecs.NewAnimationComponent("assets/sprites/gun_ss.png", utils.Vec2{X: 32, Y: 32}, gunAniStateFrames)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// gunFPComp := ecs.NewFacePositionComponent(utils.Vec2{}, true)
	//
	// bulletTraComp := ecs.NewTransformComponent(utils.Vec2{}, 1, 0)
	// bulletVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2{X: 5, Y: 0}, 0, 1)
	// bulletTimerComp, err := ecs.NewTimerComponent(500, 1, timerfuncs.Selfdestruct)
	// if err != nil {
	// 	log.Fatal("error creating bullet timer component: ", err)
	// }
	// bulletDmgComp := ecs.NewContactDamageComponent(g.playerEntity, 10, true, true, 5)
	// bulletSprComp, err := ecs.NewSpriteComponent("assets/sprites/bullet.png", 20, false)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// bulletAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	// bulletAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
	// 	{FrameIdx: 0, DurationMs: 50},
	// 	{FrameIdx: 1, DurationMs: 50},
	// 	{FrameIdx: 2, DurationMs: 50},
	// 	{FrameIdx: 1, DurationMs: 50},
	// }
	// bulletAniComp, err := ecs.NewAnimationComponent("assets/sprites/bullet_ss.png", utils.Vec2{X: 32, Y: 32}, bulletAniStateFrames)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// bulletHurtboxShape, err := shapes.NewCircleShape(3, utils.Vec2{X: 0, Y: 0})
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// bulletHurtboxComp := ecs.NewHurtboxColliderComponent(
	// 	ecs.Layer_FriendlyProjectile,
	// 	ecs.Layer_Enemy|
	// 		ecs.Layer_Terrain,
	// 	bulletHurtboxShape,
	// )
	//
	// gunSpaComp, err := ecs.NewSpawnerComponent(
	// 	utils.Vec2{X: 13, Y: 0},
	// 	ecs.SpawnerType_Point,
	// 	nil,
	// 	bulletTraComp, bulletSprComp, bulletVelComp, bulletDmgComp, bulletHurtboxComp, bulletAniComp, bulletTimerComp,
	// )
	// if err != nil {
	// 	log.Fatal("error creating gun spawner component: ", err)
	// }
	//
	// spawnName, spawnDef := abilitydefs.SpawnAbility(200)
	// gunAbis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{}
	// gunAbis[0] = ecs.EntityAbility{
	// 	Name:   spawnName,
	// 	Def:    spawnDef,
	// 	Status: ecs.AbilityStatus{State: ecs.AbiAct_Ready},
	// }
	// gunEquipmentComp := ecs.NewEquipmentComponent(ecs.Equipable_MainHand|ecs.Equipable_OffHand, gunAbis)
	//
	// gun := g.ecs.AddEntity(
	// 	gunParComp,
	// 	gunTraComp,
	// 	gunVelComp,
	// 	gunSprComp,
	// 	gunAniComp,
	// 	gunSpaComp,
	// 	gunFPComp,
	// 	gunEquipmentComp,
	// )
	//
	// pEquipperComp := ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{
	// 	ecs.Equip_MainHand: gun,
	// })
	//
	// g.ecs.AddComponent(g.playerEntity, pEquipperComp)
	//
	// err = pm.Attach(gun, g.playerEntity, g.ecs)
	// if err != nil {
	// 	log.Fatal("error attaching gun to player: ", err)
	// }

	return player, nil
}

// Spawning outside the active chunks is refused, so this has to wait until the chunks around the player are active
func SpawnStartingEquipment(ecsContainer *ecs.ECSContainer, player common.EntityId) error {
	// Bazooka
	bazookaParComp := ecs.NewHierarchyComponent()
	bazookaTraComp := ecs.NewTransformComponent(utils.Vec2f{X: 100, Y: 100}, 1, 0)
	bazookaVelComp := ecs.NewDefaultVelocityComponent()
	bazookaSprComp, err := ecs.NewSpriteComponent(common.AssetSheetBazooka, 20, true)
	if err != nil {
		return err
	}
	bazookaAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	bazookaAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 1000},
	}
	bazookaAniStateFrames[ecs.Anim_Use] = []ecs.AnimationFrame{
		{FrameIdx: 1, DurationMs: 100},
	}
	bazookaAniComp, err := ecs.NewAnimationComponent(common.AssetSheetBazooka, bazookaAniStateFrames)
	if err != nil {
		return err
	}
	bazookaFPComp := ecs.NewFacePositionComponent(utils.Vec2f{}, true)

	rocketTraComp := ecs.NewTransformComponent(utils.Vec2f{}, 1, 0)
	rocketVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2f{X: 3, Y: 0}, 0, 1)
	rocketTimerComp, err := ecs.NewTimerComponent(2000, 1, timerfuncs.TimerFunc_Selfdestruct)
	if err != nil {
		return fmt.Errorf("error creating rocket timer component: %w", err)
	}
	rocketCDComp := ecs.NewContactDamageComponent(player, 20, true, false, 65)
	rocketSprComp, err := ecs.NewSpriteComponent(common.AssetSheetRocket, 21, true)
	if err != nil {
		return err
	}
	rocketAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	rocketAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 50},
		{FrameIdx: 1, DurationMs: 50},
	}
	rocketAniComp, err := ecs.NewAnimationComponent(common.AssetSheetRocket, rocketAniStateFrames)
	if err != nil {
		return err
	}
	rocketHurtboxShape, err := shapes.NewCircleShape(5, utils.Vec2f{X: 0, Y: 0})
	if err != nil {
		return err
	}
	rocketHurtboxComp := ecs.NewHurtboxColliderComponent(
		ecs.Layer_FriendlyProjectile,
		ecs.Layer_Enemy|
			ecs.Layer_Terrain,
		rocketHurtboxShape,
	)
	explosionAniFrames := []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 50},
		{FrameIdx: 1, DurationMs: 50},
		{FrameIdx: 2, DurationMs: 50},
		{FrameIdx: 3, DurationMs: 100},
		{FrameIdx: 4, DurationMs: 100},
		{FrameIdx: 5, DurationMs: 150},
		{FrameIdx: 6, DurationMs: 200},
	}
	explodeAbi := ecs.EntityAbility{
		Def: ecs.NewAbilityDef(
			abilities.Ability_Explode,
			ecs.Ability_None,
			0,
			0,
		),
		Status: ecs.AbilityStatus{State: ecs.AbiAct_Ready},
		Params: &abilities.ExplodeParams{
			Force:           20.0,
			Radii:           []float64{10, 20, 45},
			DmgTiers:        []int{50, 25, 10},
			Animationframes: explosionAniFrames,
			SelfDestruct:    true,
		},
	}
	rocketDeathComp, err := ecs.NewDeathrattleComponent(explodeAbi)
	if err != nil {
		return fmt.Errorf("error creating rocket deathrattle component: %w", err)
	}

	bazookaSpaComp, err := ecs.NewSpawnerComponent(
		utils.Vec2f{X: 13, Y: 0},
		ecs.SpawnerType_Point,
		nil,
		rocketTraComp, rocketSprComp, rocketVelComp, rocketCDComp, rocketHurtboxComp, rocketAniComp, rocketTimerComp, rocketDeathComp,
	)
	if err != nil {
		return fmt.Errorf("error creating bazooka spawner component: %w", err)
	}

	bazookaAbis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{}
	bazookaAbis[0] = ecs.EntityAbility{
		Def: ecs.NewAbilityDef(
			abilities.Ability_Spawn,
			ecs.Ability_None,
			500,
			0,
		),
		Status: ecs.AbilityStatus{State: ecs.AbiAct_Ready},
		Params: nil,
	}
	bazookaEquipmentComp := ecs.NewEquipmentComponent(ecs.Equipable_MainHand|ecs.Equipable_OffHand, bazookaAbis)

	bazooka, err := ecsContainer.AddNewEntity(
		bazookaParComp,
		bazookaTraComp,
		bazookaVelComp,
		bazookaSprComp,
		bazookaAniComp,
		bazookaSpaComp,
		bazookaFPComp,
		bazookaEquipmentComp,
		ecs.NewPersistentComponent(),
	)
	if err != nil {
		return fmt.Errorf("error spawning bazooka: %w", err)
	}

	pEquipperComp := ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{
		ecs.Equip_MainHand: bazooka,
	})

	ecsContainer.AddComponent(player, pEquipperComp)

	err = ecsContainer.HierarchyManager.Attach(bazooka, player, ecs.Link_Held, ecsContainer)
	if err != nil {
		return fmt.Errorf("error attaching bazooka to player: %w", err)
	}

	return nil
}
