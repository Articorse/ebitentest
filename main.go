package main

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/abilitysystem"
	"ebittest/ecs/systems/animationsystem"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/ecs/systems/commonsystems"
	"ebittest/ecs/systems/damagesystem"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/ecs/systems/movementsystem"
	"ebittest/ecs/systems/platformsystem"
	"ebittest/ecs/systems/timersystem"
	"ebittest/tilesystem"
	"ebittest/utils"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	g = game{ecs: ecs.NewECSContainer(), assetmanager: assetmanager.NewAssetManager()}

	DEBUG_LEVEL                      = 0
	max_vel                          = 0.0
	prev_pos                         = utils.Vec2{}
	max_pos_diff                     = 0.0
	resolvedPhysicsCollisions uint64 = 0

	tm     = g.ecs.TransformManager
	pm     = g.ecs.ParentManager
	vm     = g.ecs.VelocityManager
	im     = g.ecs.InputManager
	pcm    = g.ecs.PhysicsColliderManager
	hurtcm = g.ecs.HurtboxColliderManager
	hitcm  = g.ecs.HitboxColliderManager

	gamepadFound = false
)

type game struct {
	ecs          *ecs.ECSContainer
	playerEntity common.EntityId

	chunkContainer *tilesystem.ChunkContainer

	assetmanager *assetmanager.AssetManager

	recording          bool
	recordingStartTick uint64
	recordedInputs     map[uint64]ecs.InputState

	replaying       bool
	replayStartTick uint64
	replayInputs    map[uint64]ecs.InputState
	replayEntity    common.EntityId
}

func (g *game) Update() error {
	var err error

	// HACK: Add gamepad here because ebiten.AppendGamepadIDs needs to be called after ebiten.RunGame
	// if !gamepadFound {
	// 	ids := ebiten.AppendGamepadIDs(nil)
	// 	id := ids[0]
	// 	for _, sId := range ids {
	// 		if ebiten.IsStandardGamepadLayoutAvailable(sId) {
	// 			id = sId
	// 			gamepadFound = true
	// 			break
	// 		}
	// 	}
	// 	if gamepadFound {
	// 		pInputConfig := make(map[ecs.InputType]ecs.InputKey)
	// 		pInputConfig[ecs.Input_Analog1Y] = ecs.NewGamepadAxisInputKey(id, ebiten.StandardGamepadAxisLeftStickVertical)
	// 		pInputConfig[ecs.Input_Analog1X] = ecs.NewGamepadAxisInputKey(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
	// 		pInputConfig[ecs.Input_Analog2Y] = ecs.NewGamepadAxisInputKey(id, ebiten.StandardGamepadAxisRightStickVertical)
	// 		pInputConfig[ecs.Input_Analog2X] = ecs.NewGamepadAxisInputKey(id, ebiten.StandardGamepadAxisRightStickHorizontal)
	// 		pInputConfig[ecs.Input_MainHandAbility1] = ecs.NewGamepadButtonInputKey(id, ebiten.StandardGamepadButtonFrontBottomRight, -1)
	// 		pInputConfig[ecs.Input_Ability1] = ecs.NewGamepadButtonInputKey(id, ebiten.StandardGamepadButtonRightBottom, -1)
	// 		pInpComp := ecs.NewInputComponent(pInputConfig, inputsources.HumanInputSource, ecs.Facing_Analog2)
	// 		g.ecs.AddComponent(g.playerEntity, pInpComp)
	// 	} else {
	// 		log.Fatal("no gamepad found")
	// 	}
	// }

	if !gamepadFound {
		pInputConfig := make(map[ecs.InputType]ecs.InputKey)
		pInputConfig[ecs.Input_Analog1Y] = ecs.NewKeyboardInputKey(ebiten.KeyS, ebiten.KeyW)
		pInputConfig[ecs.Input_Analog1X] = ecs.NewKeyboardInputKey(ebiten.KeyD, ebiten.KeyA)
		pInputConfig[ecs.Input_MainHandAbility1] = ecs.NewMouseInputKey(ebiten.MouseButtonLeft, -1)
		pInputConfig[ecs.Input_Ability1] = ecs.NewKeyboardInputKey(ebiten.KeySpace, -1)
		pInpComp := ecs.NewInputComponent(pInputConfig, ecs.InputType_Human, nil, ecs.Facing_Mouse)
		g.ecs.AddComponent(g.playerEntity, pInpComp)
		gamepadFound = true
	}

	toBeAdded, toBeRemoved, err := g.chunkContainer.GetRequiredChunks(g.ecs)
	if err != nil {
		log.Println("error populating required chunks: ", err)
	}

	err = g.chunkContainer.Tick(g.ecs.Rng, toBeAdded, toBeRemoved, g.ecs)
	if err != nil {
		log.Fatal("error generating chunk container: ", err)
	}

	tickInputs := make(map[common.EntityId]ecs.InputState)
	for _, eid := range g.ecs.Inputs.GetEntities() {
		inputType, err := im.GetInputType(eid, g.ecs)
		if err != nil {
			log.Printf("error getting input type for entity %d: %v\n", eid, err)
			continue
		}

		inputParams, err := im.GetParams(eid, g.ecs)
		if err != nil {
			log.Printf("error getting input params for entity %d: %v\n", eid, err)
			continue
		}

		inputSourceFunc, err := ecs.GetInputSourceFunc(inputType)
		if err != nil {
			log.Printf("error getting input source func for entity %d: %v\n", eid, err)
			continue
		}

		tInputs, err := inputSourceFunc(eid, g.ecs.TickIdx, inputParams, g.ecs)
		if err != nil {
			log.Printf("error getting tick inputs for entity %d: %v\n", eid, err)
			continue
		}

		tickInputs[eid] = tInputs
	}
	g.ecs.SetTickInputs(g.ecs.TickIdx, tickInputs)

	currentTickInputs, err := g.ecs.GetCurrentTickInputs()
	if err != nil {
		log.Printf("error getting current tick inputs: %v\n", err)
	}
	err = inputsystem.HandleInputs(
		g.ecs.Camera,
		g.ecs,
		currentTickInputs,
	)
	if err != nil {
		log.Printf("error during handling inputs: %v", err)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		f, err := os.Create("replay.json")
		if err != nil {
			log.Println("error creating replay file: ", err)
		}
		defer f.Close()

		j, err := json.Marshal(g.ecs.InputLog)
		if err != nil {
			log.Println("error marshalling replay log: ", err)
		}

		_, err = f.Write(j)
		if err != nil {
			log.Println("error writing replay file: ", err)
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.ecs.Camera.X -= 10
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.ecs.Camera.Y -= 10
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.ecs.Camera.X += 10
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.ecs.Camera.Y += 10
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.ecs.CameraFollow = !g.ecs.CameraFollow
	}
	if g.ecs.CameraFollow {
		pWorldPos, err := tm.GetWorldPos(g.playerEntity, g.ecs)
		if err != nil {
			log.Println("error getting player ecs position for camera follow: ", err)
		}

		g.ecs.Camera = pWorldPos.Subtract(utils.Vec2{X: float64(data.CameraWidth) / 2, Y: float64(data.CameraHeight) / 2})
	}

	// Replay
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		if !g.recording {
			g.recording = true
			g.recordingStartTick = g.ecs.TickIdx
			g.recordedInputs = make(map[uint64]ecs.InputState)
		} else {
			g.recording = false
			// TODO: Save to file
		}
	}
	if g.recording {
		relTick := g.ecs.TickIdx - g.recordingStartTick
		g.recordedInputs[relTick] = g.ecs.InputLog[g.ecs.TickIdx][g.playerEntity]
	}

	// DEBUG
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		DEBUG_LEVEL++
		if DEBUG_LEVEL > 2 {
			DEBUG_LEVEL = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		max_vel = 0
		max_pos_diff = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		parEnt := pm.GetEntity(common.EntityId(1), g.ecs)

		if parEnt > -1 {
			err = pm.Detach(common.EntityId(1), g.ecs)
			if err != nil {
				log.Println("error detaching gun: ", err)
			}
		} else {
			err = pm.Attach(common.EntityId(1), g.playerEntity, g.ecs)
			if err != nil {
				log.Println("error attaching gun to player: ", err)
			}
		}
	}

	pWorldPos, err := tm.GetWorldPos(g.playerEntity, g.ecs)
	if err != nil {
		log.Println("error getting player ecs position for debug: ", err)
	} else {
		pos_diff := prev_pos.Subtract(pWorldPos).Length()
		if max_pos_diff < pos_diff {
			max_pos_diff = pos_diff
		}
		prev_pos = pWorldPos
	}

	// END DEBUG

	err = animationsystem.Tick(g.ecs, g.assetmanager)
	if err != nil {
		log.Println("animation system tick error: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		}
	}

	err = timersystem.Tick(g.ecs)
	if err != nil {
		log.Println("timer system tick error: ", err)
	}
	g.ecs.TickState = *common.NewTickState()

	if err := movementsystem.Tick(g.ecs); err != nil {
		log.Println("movement system error: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		}
	}

	g.ecs.TickState.CollisionGrid, err = commonsystems.PopulateSpatialHashGrid(g.ecs, data.SpatialHashGridCellSize)
	if err != nil {
		log.Println("error during populating spatial hash grid: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		}
	}

	proximateEntities, err := commonsystems.GetSHGProximities(g.ecs.TickState.CollisionGrid, g.ecs)
	if err != nil {
		log.Println("error during spatial hash grid proximity checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		}
	}

	physicsAABBCollisions, err := collisionsystem.GetAABBCollisions(pcm, pcm, proximateEntities, g.ecs)
	if err != nil {
		log.Println("error during AABB collision checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		}
	}

	physicsCollisions, err := collisionsystem.GetCollisions(pcm, pcm, physicsAABBCollisions, g.ecs)
	if err != nil {
		log.Println("error during physics collision checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		}
	}

	resolvedPhysicsCollisions, err = collisionsystem.ResolvePhysicsCollisions(physicsCollisions, g.ecs)
	if err != nil {
		log.Println("error during collision resolution: ", err)
	}

	potentialCollisionTiles, err := g.chunkContainer.GetTilesWithPotentialCollisions(g.ecs, data.TileSize)
	if err != nil {
		log.Println("error during getting tiles with potential collisions: ", err)
	}

	tileAABBCollisions, err := tilesystem.GetAABBCollisions(potentialCollisionTiles, g.ecs)
	if err != nil {
		log.Println("error during tile AABB collision checking: ", err)
	}

	tileCollisions, err := tilesystem.GetCollisions(tileAABBCollisions, g.ecs)
	if err != nil {
		log.Println("error during tile collision checking: ", err)
	}

	_, err = tilesystem.ResolveTileCollisions(tileCollisions, g.ecs)
	if err != nil {
		log.Println("error during tile collision resolution: ", err)
	}

	err = abilitysystem.Tick(g.ecs)
	if err != nil {
		log.Println("ability system tick error: ", err)
	}

	damageAABBCollisions, err := collisionsystem.GetMirrorAABBCollisions(hurtcm, hitcm, proximateEntities, g.ecs)
	if err != nil {
		log.Println("error during AABB damage collision checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		}
	}

	damageCollisions, err := collisionsystem.GetCollisions(hurtcm, hitcm, damageAABBCollisions, g.ecs)
	if err != nil {
		log.Println("error during damage collision checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		}
	}

	damagesystem.Tick(g.ecs)

	_, err = damagesystem.DealContactDamage(damageCollisions, g.ecs)
	if err != nil {
		log.Println("error during dealing contact damage: ", err)
	}

	err = platformsystem.Tick(g.ecs.TickState.CollisionGrid, g.ecs)
	if err != nil {
		log.Println("platform system tick error: ", err)
	}

	if err := g.ecs.RemoveScheduledEntities(); err != nil {
		log.Println("error while removing scheduled entities: ", err)
	}

	g.ecs.TickState.ProximateEntities = proximateEntities
	g.ecs.TickState.AABBCollisions = physicsAABBCollisions
	g.ecs.TickState.Collisions = physicsCollisions

	g.ecs.TickIdx++

	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	if err := tilesystem.DrawChunks(screen, g.ecs.Camera, g.chunkContainer); err != nil {
		log.Println("error while drawing chunks: ", err)
	}

	if err := drawsystem.DrawSprites(
		screen,
		g.ecs.Camera,
		g.ecs.TickState.CollisionGrid,
		g.ecs,
		g.assetmanager,
	); err != nil {
		log.Println("error while drawing frame, removing offending entity: ", err)
		var missingDependencyError *common.ErrorMissingComponentDependency
		if errors.As(err, &missingDependencyError) {
			g.ecs.ScheduleRemoveEntity(missingDependencyError.Entity)
		}
		var missingExpectedComponentError *common.ErrorMissingExpectedComponent
		if errors.As(err, &missingExpectedComponentError) {
			g.ecs.ScheduleRemoveEntity(missingExpectedComponentError.Entity)
		}
	}

	if err := drawsystem.DrawFloatingTexts(screen, g.ecs); err != nil {
		log.Println("error while drawing floating texts: ", err)
	}

	g.DrawDebug(screen)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return data.CameraWidth, data.CameraHeight
}

func (g *game) DrawDebug(screen *ebiten.Image) {
	if DEBUG_LEVEL == 1 {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %f", ebiten.ActualFPS()))
	}

	if DEBUG_LEVEL == 2 {
		if err := collisionsystem.DrawColliders(pcm, data.Debug_ColliderColor, data.Debug_ColliderCollidedColor, screen, g.ecs.Camera, g.ecs.TickState.Collisions, g.ecs); err != nil {
			log.Println("error while drawing physics colliders: ", err, "removing entity")
			var invalidComponentsErr *common.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
			}
		}

		// if err := collisionsystem.DrawColliders(hurtcm, data.Debug_ColliderColor, data.Debug_ColliderCollidedColor, screen, g.ecs.Camera, g.ecs.TickState.Collisions, g.ecs); err != nil {
		// 	log.Println("error while drawing physics colliders: ", err, "removing entity")
		// 	var invalidComponentsErr *common.ErrorMissingComponentDependency
		// 	if errors.As(err, &invalidComponentsErr) {
		// 		g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		// 	}
		// }

		// if err := collisionsystem.DrawColliders(hitcm, data.Debug_ColliderColor, data.Debug_ColliderCollidedColor, screen, g.ecs.Camera, g.ecs.TickState.Collisions, g.ecs); err != nil {
		// 	log.Println("error while drawing physics colliders: ", err, "removing entity")
		// 	var invalidComponentsErr *common.ErrorMissingComponentDependency
		// 	if errors.As(err, &invalidComponentsErr) {
		// 		g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		// 	}
		// }

		if err := collisionsystem.DrawAABBs(pcm, data.Debug_AABBColliderColor, data.Debug_AABBColliderColor, screen, g.ecs.Camera, g.ecs.TickState.AABBCollisions, g.ecs); err != nil {
			log.Println("error while drawing AABBs: ", err, "removing entity")
			var invalidComponentsErr *common.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
			}
		}

		// if err := collisionsystem.DrawAABBs(hurtcm, data.Debug_AABBColliderColor, data.Debug_AABBColliderColor, screen, g.ecs.Camera, g.ecs.TickState.AABBCollisions, g.ecs); err != nil {
		// 	log.Println("error while drawing AABBs: ", err, "removing entity")
		// 	var invalidComponentsErr *common.ErrorMissingComponentDependency
		// 	if errors.As(err, &invalidComponentsErr) {
		// 		g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		// 	}
		// }
		//
		// if err := collisionsystem.DrawAABBs(hitcm, data.Debug_AABBColliderColor, data.Debug_AABBColliderColor, screen, g.ecs.Camera, g.ecs.TickState.AABBCollisions, g.ecs); err != nil {
		// 	log.Println("error while drawing AABBs: ", err, "removing entity")
		// 	var invalidComponentsErr *common.ErrorMissingComponentDependency
		// 	if errors.As(err, &invalidComponentsErr) {
		// 		g.ecs.ScheduleRemoveEntity(invalidComponentsErr.Entity)
		// 	}
		// }

		if err := collisionsystem.DrawCollisions(screen, data.Debug_CollisionVectorColor, g.ecs.Camera, g.ecs.TickState.Collisions, g.ecs); err != nil {
			log.Println("error while drawing collisions: ", err)
		}

		proximateEntitiesCount := 0
		for _, others := range g.ecs.TickState.ProximateEntities {
			for range others {
				proximateEntitiesCount++
			}
		}

		pLocalVelVec, err := vm.GetLocalVector(g.playerEntity, g.ecs)
		if err != nil {
			log.Fatalf("error getting player local velocity for debug: %v", err)
		}

		vel := pLocalVelVec.Length()
		if vel > max_vel {
			max_vel = vel
		}

		ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %f\nTick: %d\nVel: %v\nMaxVel: %f\nMaxPosDiff: %v\nSHG Cells: %v\nProximate Pairs: %d\nResolved Collisions: %d", ebiten.ActualFPS(), g.ecs.TickIdx, pLocalVelVec, max_vel, max_pos_diff, g.ecs.TickState.CollisionGrid, proximateEntitiesCount, resolvedPhysicsCollisions))
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(data.TPS)

	g.chunkContainer = &tilesystem.ChunkContainer{}
	err := g.chunkContainer.GenerateTileAtlasFromJson("assets/tiles/atlas.json")
	if err != nil {
		log.Fatal("error generating tile atlas from json: ", err)
	}

	g.ecs.TickState = *common.NewTickState()

	// Player
	pParComp := ecs.NewParentComponent()
	pTraComp := ecs.NewTransformComponent(utils.Vec2{X: 100, Y: 100}, 1, 0)
	pVelComp := ecs.NewDefaultVelocityComponent()
	pSprComp, err := ecs.NewSpriteComponent(common.AssetSheetSlime, 20, false)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	pPhyColShape, err := shapes.NewCircleShape(5, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
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
	pHitboxShape, err := shapes.NewCircleShape(5, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
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
			ecs.Ability_Dodge,
			ecs.Ability_Dodge_Post,
			1000,
			200,
		),
		Status: ecs.AbilityStatus{State: ecs.AbiAct_Ready},
		Params: ecs.DodgeParams{
			Force: 10,
		},
	}
	pAbiComp := ecs.NewAbilitiesComponent(pAbis)
	pCLComp := ecs.NewChunkLoaderComponent(1)
	g.playerEntity = g.ecs.AddEntity(
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
	)

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

	// Bazooka
	bazookaParComp := ecs.NewParentComponent()
	bazookaTraComp := ecs.NewTransformComponent(utils.Vec2{X: 100, Y: 100}, 1, 0)
	bazookaVelComp := ecs.NewDefaultVelocityComponent()
	bazookaSprComp, err := ecs.NewSpriteComponent(common.AssetSheetBazooka, 20, true)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	bazookaFPComp := ecs.NewFacePositionComponent(utils.Vec2{}, true)

	rocketTraComp := ecs.NewTransformComponent(utils.Vec2{}, 1, 0)
	rocketVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2{X: 3, Y: 0}, 0, 1)
	rocketTimerComp, err := ecs.NewTimerComponent(2000, 1, ecs.TimerFunc_Selfdestruct)
	if err != nil {
		log.Fatal("error creating rocket timer component: ", err)
	}
	rocketCDComp := ecs.NewContactDamageComponent(g.playerEntity, 20, true, false, 65)
	rocketSprComp, err := ecs.NewSpriteComponent(common.AssetSheetRocket, 21, true)
	if err != nil {
		log.Fatal(err)
	}
	rocketAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	rocketAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 50},
		{FrameIdx: 1, DurationMs: 50},
	}
	rocketAniComp, err := ecs.NewAnimationComponent(common.AssetSheetRocket, rocketAniStateFrames)
	if err != nil {
		log.Fatal(err)
	}
	rocketHurtboxShape, err := shapes.NewCircleShape(5, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
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
			ecs.Ability_Explode,
			ecs.Ability_None,
			0,
			0,
		),
		Status: ecs.AbilityStatus{State: ecs.AbiAct_Ready},
		Params: ecs.ExplodeParams{
			Force:           20.0,
			Radii:           []float64{10, 20, 45},
			DmgTiers:        []int{50, 25, 10},
			Animationframes: explosionAniFrames,
			SelfDestruct:    true,
		},
	}
	rocketDeathComp, err := ecs.NewDeathrattleComponent(explodeAbi)
	if err != nil {
		log.Fatal("error creating rocket deathrattle component: ", err)
	}

	bazookaSpaComp, err := ecs.NewSpawnerComponent(
		utils.Vec2{X: 13, Y: 0},
		ecs.SpawnerType_Point,
		nil,
		rocketTraComp, rocketSprComp, rocketVelComp, rocketCDComp, rocketHurtboxComp, rocketAniComp, rocketTimerComp, rocketDeathComp,
	)
	if err != nil {
		log.Fatal("error creating bazooka spawner component: ", err)
	}

	bazookaAbis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{}
	bazookaAbis[0] = ecs.EntityAbility{
		Def: ecs.NewAbilityDef(
			ecs.Ability_Spawn,
			ecs.Ability_None,
			500,
			0,
		),
		Status: ecs.AbilityStatus{State: ecs.AbiAct_Ready},
		Params: nil,
	}
	bazookaEquipmentComp := ecs.NewEquipmentComponent(ecs.Equipable_MainHand|ecs.Equipable_OffHand, bazookaAbis)

	bazooka := g.ecs.AddEntity(
		bazookaParComp,
		bazookaTraComp,
		bazookaVelComp,
		bazookaSprComp,
		bazookaAniComp,
		bazookaSpaComp,
		bazookaFPComp,
		bazookaEquipmentComp,
	)

	pEquipperComp := ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{
		ecs.Equip_MainHand: bazooka,
	})

	g.ecs.AddComponent(g.playerEntity, pEquipperComp)

	err = pm.Attach(bazooka, g.playerEntity, g.ecs)
	if err != nil {
		log.Fatal("error attaching bazooka to player: ", err)
	}

	// // Enemy spawner
	// enemyTraComp := ecs.NewTransformComponent(utils.Vec2{X: 300, Y: 150}, 1, 0)
	// enemyVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2{}, data.DefaultAcceleration*0.25, data.DefaultDrag)
	// enemyParComp := ecs.NewParentComponent()
	// enemySprComp, err := ecs.NewSpriteComponent(common.AssetSheetEvilSlime, 20, true)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// enemyAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	// enemyAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
	// 	{FrameIdx: 0, DurationMs: 2000},
	// 	{FrameIdx: 1, DurationMs: 100},
	// 	{FrameIdx: 2, DurationMs: 100},
	// 	{FrameIdx: 3, DurationMs: 100},
	// 	{FrameIdx: 4, DurationMs: 100},
	// 	{FrameIdx: 3, DurationMs: 100},
	// 	{FrameIdx: 2, DurationMs: 100},
	// 	{FrameIdx: 1, DurationMs: 100},
	// }
	// enemyAniComp, err := ecs.NewAnimationComponent(
	// 	common.AssetSheetEvilSlime,
	// 	enemyAniStateFrames,
	// )
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// enemyHpComp := ecs.NewHitpointsComponent(20, 100)
	// enemyPhyColShape, err := shapes.NewCircleShape(7, utils.Vec2{X: 0, Y: 0})
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// enemyPhyColComp := ecs.NewPhysicsColliderComponent(
	// 	ecs.Layer_Enemy,
	// 	ecs.Layer_Player|
	// 		ecs.Layer_FriendlyProjectile|
	// 		ecs.Layer_Platform|
	// 		ecs.Layer_Enemy|
	// 		ecs.Layer_Terrain,
	// 	ecs.Collider_Mob,
	// 	enemyPhyColShape,
	// )
	// enemyHitboxColShape, err := shapes.NewCircleShape(5, utils.Vec2{X: 0, Y: 0})
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// enemyHitboxComp := ecs.NewHitboxColliderComponent(
	// 	ecs.Layer_Enemy,
	// 	ecs.Layer_Player|
	// 		ecs.Layer_FriendlyProjectile|
	// 		ecs.Layer_Platform|
	// 		ecs.Layer_Terrain,
	// 	enemyHitboxColShape,
	// )
	// enemyHurtboxColShape, err := shapes.NewCircleShape(7, utils.Vec2{X: 0, Y: 0})
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// enemyHurtboxComp := ecs.NewHurtboxColliderComponent(
	// 	ecs.Layer_Enemy,
	// 	ecs.Layer_Player|
	// 		ecs.Layer_FriendlyProjectile|
	// 		ecs.Layer_Platform|
	// 		ecs.Layer_Terrain,
	// 	enemyHurtboxColShape,
	// )
	// enemyCDmgComp := ecs.NewContactDamageComponent(-1, 20, false, false, 1)
	// enemyFollowInputParams := ecs.InputFollowParams{
	// 	FollowEntity: g.playerEntity,
	// }
	// enemyInputComp := ecs.NewInputComponent(nil, ecs.InputType_Follow, enemyFollowInputParams, ecs.Facing_None)
	//
	// enemySpawnerTimerComp, err := ecs.NewTimerComponent(1000, -1, ecs.TimerFunc_Spawn)
	// if err != nil {
	// 	log.Fatal("error creating enemy spawner timer component: ", err)
	// }
	// enemySpawnerShape, err := shapes.NewRectangleShape(data.CameraWidth+100, data.CameraHeight+100, utils.Vec2{})
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// enemySpawnerComp, err := ecs.NewSpawnerComponent(
	// 	utils.Vec2{
	// 		X: g.ecs.Camera.X + (data.CameraWidth / 2),
	// 		Y: g.ecs.Camera.Y + (data.CameraHeight / 2),
	// 	},
	// 	ecs.SpawnerType_Perimeter,
	// 	enemySpawnerShape,
	// 	enemyParComp,
	// 	enemyTraComp,
	// 	enemyVelComp,
	// 	enemySprComp,
	// 	enemyAniComp,
	// 	enemyHpComp,
	// 	enemyPhyColComp,
	// 	enemyHitboxComp,
	// 	enemyHurtboxComp,
	// 	enemyCDmgComp,
	// 	enemyInputComp,
	// )
	// if err != nil {
	// 	log.Fatal("error creating enemy spawner component: ", err)
	// }
	// g.ecs.AddEntity(enemySpawnerTimerComp, enemySpawnerComp)

	// Tree
	treeInpComp := ecs.NewInputComponent(nil, ecs.InputType_Dummy, nil, ecs.Facing_None)
	treeParComp := ecs.NewParentComponent()
	treeTraComp := ecs.NewTransformComponent(utils.Vec2{X: 200, Y: 200}, 1, 0)
	treeVelComp := ecs.NewDefaultVelocityComponent()
	treeSprComp, err := ecs.NewSpriteComponent(common.AssetImageTree, 20, true)
	if err != nil {
		log.Fatal(err)
	}

	treeColShape, err := shapes.NewRectangleShape(10, 5, utils.Vec2{X: -1, Y: 9})
	// treeColShape, err := shapes.NewCircleShape(5, utils.Vec2{X: -1, Y: 9})
	if err != nil {
		log.Fatal(err)
	}
	treeColComp := ecs.NewPhysicsColliderComponent(
		ecs.Layer_Terrain,
		ecs.Layer_Player|
			ecs.Layer_Enemy|
			ecs.Layer_EnemyProjectile|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Platform,
		ecs.Collider_Static,
		treeColShape,
	)

	tree := g.ecs.AddEntity(
		treeInpComp,
		treeParComp,
		treeTraComp,
		treeVelComp,
		treeSprComp,
		treeColComp,
	)

	g.replayEntity = tree

	// Platform
	var inputLoop []ecs.InputState
	for range 100 {
		inputLoop = append(inputLoop, ecs.InputState{Analog1X: -1})
	}
	for range 100 {
		inputLoop = append(inputLoop, ecs.InputState{Analog1X: 1})
	}
	loopParams := &ecs.InputLoopParams{
		LoopInputs: inputLoop,
	}
	platInput := ecs.NewInputComponent(nil, ecs.InputType_Loop, loopParams, ecs.Facing_None)

	platParComp := ecs.NewParentComponent()
	platTraComp := ecs.NewTransformComponent(utils.Vec2{X: 100, Y: 10}, 1, 0)
	platVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2{}, 0.3, data.DefaultDrag)

	platSprComp, err := ecs.NewSpriteComponent(common.AssetImagePlatform, 10, true)
	if err != nil {
		log.Fatal(err)
	}

	platColShape, err := shapes.NewRectangleShape(28, 28, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
	}
	platColComp := ecs.NewPlatformColliderComponent(
		ecs.Layer_Platform,
		ecs.Layer_Player|
			ecs.Layer_Enemy|
			ecs.Layer_EnemyProjectile|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Terrain,
		[]shapes.Shape{platColShape},
	)

	_ = g.ecs.AddEntity(
		platInput,
		platParComp,
		platTraComp,
		platVelComp,
		platSprComp,
		platColComp,
	)

	// for _ = range 500 {
	// 	g.AddRandomEntity()
	// }

	ebiten.SetWindowSize(1920, 1080)
	ebiten.SetWindowTitle("ebittest")

	if err := ebiten.RunGame(&g); err != nil {
		log.Fatal(err)
	}
}

func (g *game) AddRandomEntity() {
	x := g.ecs.Rng.IntN(10000)
	y := g.ecs.Rng.IntN(10000)

	traComp := ecs.NewTransformComponent(utils.Vec2{X: float64(x), Y: float64(y)}, 1, 0)
	sprComp, err := ecs.NewSpriteComponent(common.AssetImageTree, 20, true)
	if err != nil {
		log.Fatal(err)
	}

	velComp := ecs.NewDefaultVelocityComponent()
	shape, err := shapes.NewRectangleShape(10, 5, utils.Vec2{X: -1, Y: 9})
	colComp := ecs.NewPhysicsColliderComponent(
		ecs.Layer_Terrain,
		ecs.Layer_Player|
			ecs.Layer_Enemy|
			ecs.Layer_EnemyProjectile|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Platform,
		ecs.Collider_Static,
		shape,
	)

	_ = g.ecs.AddEntity(
		traComp,
		velComp,
		sprComp,
		colComp,
	)
}
