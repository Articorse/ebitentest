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
	"math"
)

func SpawnPlayer(ecsContainer *ecs.ECSContainer) (common.EntityId, error) {
	// Player
	pParComp := ecs.NewHierarchyComponent()
	pTraComp := ecs.NewTransformComponent(utils.Vec2f{}, 1, 0)
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
			ecs.Layer_Water|
			ecs.Layer_Feature|
			ecs.Layer_Platform,
		ecs.Collider_Mob,
		pPhyColShape,
	)
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
	pCLComp := ecs.NewChunkLoaderComponent(2)
	player, err := ecsContainer.AddNewEntity(
		pParComp,
		pTraComp,
		pVelComp,
		pSprComp,
		pAniComp,
		pPhyColComp,
		ecs.NewBodyComponent(data.Body_Humanoid),
		ecs.NewFactionComponent(data.Faction_Colony),
		ecs.NewInvulnerabilityComponent(1000),
		pHitboxComp,
		pAbiComp,
		pCLComp,
		ecs.NewPersistentComponent(),
		ecs.NewVisionComponent(data.PlayerVisionRangeTiles, 1),
		ecs.NewInventoryComponent(data.InventoryRows*data.InventoryColumns, data.HotbarSlots),
		ecs.NewAimComponent(),
		ecs.NewConstructorComponent(data.PlayerBuildRangeTiles, data.PlayerBuildWork, data.PlayerBuildCooldownMs),
	)
	if err != nil {
		return -1, fmt.Errorf("error spawning player: %w", err)
	}

	return player, nil
}

// Spawning outside the active chunks is refused, so this has to wait until the chunks around the player are active
func SpawnStartingEquipment(ecsContainer *ecs.ECSContainer, player common.EntityId) error {
	// Bazooka
	bazookaParComp := ecs.NewHierarchyComponent()
	playerPos, err := ecsContainer.WorldPos(player)
	if err != nil {
		return fmt.Errorf("error getting world position of player %d: %w", player, err)
	}
	bazookaTraComp := ecs.NewTransformComponent(playerPos, 1, 0)
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
	bazookaFacingComp := ecs.NewFacingComponent().WithTurnRate(2 * math.Pi)

	rocketTraComp := ecs.NewTransformComponent(utils.Vec2f{}, 1, 0)
	rocketVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2f{X: 3, Y: 0}, 0, 1)
	rocketTimerComp, err := ecs.NewTimerComponent(2000, 1, timerfuncs.TimerFunc_Selfdestruct)
	if err != nil {
		return fmt.Errorf("error creating rocket timer component: %w", err)
	}
	rocketCDComp := ecs.NewContactDamageComponent(player, 20, 65).WithDieOnContact()
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
			ecs.Layer_Terrain|
			ecs.Layer_Feature,
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
		bazookaFacingComp,
		bazookaEquipmentComp,
		ecs.NewItemComponent(),
		ecs.NewPersistentComponent(),
	)
	if err != nil {
		return fmt.Errorf("error spawning bazooka: %w", err)
	}

	ecsContainer.AddComponent(player, ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{}))

	mattock, err := spawnMattock(ecsContainer, playerPos)
	if err != nil {
		return fmt.Errorf("error spawning mattock: %w", err)
	}
	if err := ecsContainer.PutInInventory(player, 0, mattock); err != nil {
		return fmt.Errorf("error putting mattock in the player's inventory: %w", err)
	}

	if err := ecsContainer.PutInInventory(player, 1, bazooka); err != nil {
		return fmt.Errorf("error putting bazooka in the player's inventory: %w", err)
	}

	gun, err := spawnGun(ecsContainer, player, playerPos)
	if err != nil {
		return fmt.Errorf("error spawning gun: %w", err)
	}
	if err := ecsContainer.PutInInventory(player, 2, gun); err != nil {
		return fmt.Errorf("error putting gun in the player's inventory: %w", err)
	}

	axe, err := spawnAxe(ecsContainer, playerPos)
	if err != nil {
		return fmt.Errorf("error spawning axe: %w", err)
	}
	if err := ecsContainer.PutInInventory(player, 3, axe); err != nil {
		return fmt.Errorf("error putting axe in the player's inventory: %w", err)
	}

	torches, err := ecsContainer.SpawnItemStack(data.Item_Torch, 100, playerPos)
	if err != nil {
		return fmt.Errorf("error spawning torches: %w", err)
	}
	if err := ecsContainer.PutInInventory(player, 4, torches); err != nil {
		return fmt.Errorf("error putting torches in the player's inventory: %w", err)
	}

	for i, item := range []common.EntityId{mattock, bazooka, gun, axe, torches} {
		if err := ecsContainer.BindHotbar(player, i, item); err != nil {
			return fmt.Errorf("error putting item %d on the player's hotbar: %w", item, err)
		}
	}
	if err := ecsContainer.SelectHotbarSlot(player, 0); err != nil {
		return fmt.Errorf("error selecting the player's first hotbar slot: %w", err)
	}

	return nil
}

func spawnMattock(ecsContainer *ecs.ECSContainer, pos utils.Vec2f) (common.EntityId, error) {
	sprComp, err := ecs.NewSpriteComponent(common.AssetImageMattock, 20, false)
	if err != nil {
		return -1, err
	}

	abis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{}
	abis[0] = ecs.EntityAbility{
		Def:    ecs.NewAbilityDef(abilities.Ability_TileStrike, ecs.Ability_None, 400, 0),
		Status: ecs.AbilityStatus{State: ecs.AbiAct_Ready},
		Params: &abilities.TileStrikeParams{
			RangeTiles: 3,
			Damage:     50,
			Mask:       ecs.Layer_Terrain | ecs.Layer_Ground | ecs.Layer_Feature,
		},
	}

	return ecsContainer.AddNewEntity(
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(pos, 1, 0),
		ecs.NewDefaultVelocityComponent(),
		sprComp,
		ecs.NewEquipmentComponent(ecs.Equipable_MainHand|ecs.Equipable_OffHand, abis),
		ecs.NewSwingComponent(2*math.Pi/3, 0.15, utils.Vec2f{X: 15.5, Y: 21.5}),
		ecs.NewItemComponent(),
		ecs.NewPersistentComponent(),
	)
}

func spawnAxe(ecsContainer *ecs.ECSContainer, pos utils.Vec2f) (common.EntityId, error) {
	sprComp, err := ecs.NewSpriteComponent(common.AssetImageAxe, 20, false)
	if err != nil {
		return -1, err
	}

	abis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{}
	abis[0] = ecs.EntityAbility{
		Def:    ecs.NewAbilityDef(abilities.Ability_Slash, ecs.Ability_None, 600, 0),
		Status: ecs.AbilityStatus{State: ecs.AbiAct_Ready},
		Params: &abilities.SlashParams{
			Image: common.AssetImageSlash,
			// Around the crescent in slash.png, which is 20×36
			Hull: []utils.Vec2f{
				{X: -10, Y: -18}, {X: -7, Y: -18}, {X: -1, Y: -16}, {X: 1, Y: -15}, {X: 8, Y: -8}, {X: 9, Y: -6}, {X: 10, Y: -3},
				{X: 10, Y: 3}, {X: 9, Y: 6}, {X: 8, Y: 8}, {X: 1, Y: 15}, {X: -1, Y: 16}, {X: -7, Y: 18}, {X: -10, Y: 18},
			},
			ReachPx:    14,
			Damage:     50,
			Mask:       ecs.Layer_Enemy | ecs.Layer_Feature,
			DurationMs: 150,
		},
	}

	return ecsContainer.AddNewEntity(
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(pos, 1, 0),
		ecs.NewDefaultVelocityComponent(),
		sprComp,
		ecs.NewEquipmentComponent(ecs.Equipable_MainHand|ecs.Equipable_OffHand, abis),
		ecs.NewSwingComponent(2*math.Pi/3, 0.15, utils.Vec2f{X: 15.5, Y: 21.5}),
		ecs.NewItemComponent(),
		ecs.NewPersistentComponent(),
	)
}

func spawnGun(ecsContainer *ecs.ECSContainer, player common.EntityId, pos utils.Vec2f) (common.EntityId, error) {
	gunSprComp, err := ecs.NewSpriteComponent(common.AssetSheetGun, 20, true)
	if err != nil {
		return -1, err
	}
	gunAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	gunAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 1000},
	}
	gunAniStateFrames[ecs.Anim_Use] = []ecs.AnimationFrame{
		{FrameIdx: 1, DurationMs: 100},
	}
	gunAniComp, err := ecs.NewAnimationComponent(common.AssetSheetGun, gunAniStateFrames)
	if err != nil {
		return -1, err
	}

	bulletTimerComp, err := ecs.NewTimerComponent(500, 1, timerfuncs.TimerFunc_Selfdestruct)
	if err != nil {
		return -1, fmt.Errorf("error creating bullet timer component: %w", err)
	}
	bulletDmgComp := ecs.NewContactDamageComponent(player, 10, 5).WithDieOnContact()
	bulletSprComp, err := ecs.NewSpriteComponent(common.AssetSheetBullet, 20, false)
	if err != nil {
		return -1, err
	}
	bulletAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	bulletAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 50},
		{FrameIdx: 1, DurationMs: 50},
		{FrameIdx: 2, DurationMs: 50},
		{FrameIdx: 1, DurationMs: 50},
	}
	bulletAniComp, err := ecs.NewAnimationComponent(common.AssetSheetBullet, bulletAniStateFrames)
	if err != nil {
		return -1, err
	}
	bulletHurtboxShape, err := shapes.NewCircleShape(3, utils.Vec2f{X: 0, Y: 0})
	if err != nil {
		return -1, err
	}
	bulletHurtboxComp := ecs.NewHurtboxColliderComponent(
		ecs.Layer_FriendlyProjectile,
		ecs.Layer_Enemy|
			ecs.Layer_Terrain|
			ecs.Layer_Feature,
		bulletHurtboxShape,
	)

	gunSpaComp, err := ecs.NewSpawnerComponent(
		utils.Vec2f{X: 13, Y: 0},
		ecs.SpawnerType_Point,
		nil,
		ecs.NewTransformComponent(utils.Vec2f{}, 1, 0),
		bulletSprComp,
		ecs.NewVelocityComponentWithParams(utils.Vec2f{X: 5, Y: 0}, 0, 1),
		bulletDmgComp,
		bulletHurtboxComp,
		bulletAniComp,
		bulletTimerComp,
	)
	if err != nil {
		return -1, fmt.Errorf("error creating gun spawner component: %w", err)
	}

	gunAbis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{}
	gunAbis[0] = ecs.EntityAbility{
		Def:    ecs.NewAbilityDef(abilities.Ability_Spawn, ecs.Ability_None, 200, 0),
		Status: ecs.AbilityStatus{State: ecs.AbiAct_Ready},
	}

	return ecsContainer.AddNewEntity(
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(pos, 1, 0),
		ecs.NewDefaultVelocityComponent(),
		gunSprComp,
		gunAniComp,
		gunSpaComp,
		ecs.NewFacingComponent(),
		ecs.NewEquipmentComponent(ecs.Equipable_MainHand|ecs.Equipable_OffHand, gunAbis),
		ecs.NewItemComponent(),
		ecs.NewPersistentComponent(),
	)
}
