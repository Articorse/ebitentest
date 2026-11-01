package main

import (
	"ebittest/assetmanager"
	"ebittest/content"
	"ebittest/content/inputsources"
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
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
	"ebittest/world"
	"errors"
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	g = game{ecs: ecs.NewECSContainer(), assetmanager: assetmanager.NewAssetManager()}

	DEBUG_LEVEL                      = 0
	max_vel                          = 0.0
	prev_pos                         = utils.Vec2f{}
	max_pos_diff                     = 0.0
	resolvedPhysicsCollisions uint64 = 0

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
	loading        bool
	justLoaded     bool
	newWorld       bool

	assetmanager *assetmanager.AssetManager
}

func (g *game) Update() error {
	var err error

	if ebiten.IsWindowBeingClosed() {
		return g.saveAndExit()
	}

	if g.loading {
		ready, err := g.chunkContainer.Prepare(g.ecs)
		if err != nil {
			log.Println("error preparing chunks:", err)
		}

		g.loading = !ready
		g.justLoaded = ready
		return nil
	}

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
		pInpComp := ecs.NewInputComponent(pInputConfig, inputsources.InputType_Human, nil, ecs.Facing_Mouse)
		g.ecs.AddComponent(g.playerEntity, pInpComp)
		gamepadFound = true
	}

	err = g.chunkContainer.Tick(g.ecs)
	if err != nil {
		log.Println("error ticking chunk container:", err)
	}

	// The first tick activates the chunks loaded during loading, so the whole world can be checked only now
	if g.justLoaded {
		g.justLoaded = false
		if g.newWorld {
			g.newWorld = false
			if err := templates.SpawnStartingEquipment(g.ecs, g.playerEntity); err != nil {
				log.Fatal("error spawning starting equipment: ", err)
			}
		}
		g.logConsistency("after loading")
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

		inputSourceFunc, err := im.GetInputSourceFunc(inputType, g.ecs)
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
		pWorldPos, err := g.ecs.WorldPos(g.playerEntity)
		if err != nil {
			log.Println("error getting player ecs position for camera follow: ", err)
		}

		g.ecs.Camera = pWorldPos.Subtract(utils.Vec2f{X: float64(data.CameraWidth) / 2, Y: float64(data.CameraHeight) / 2})
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
		parEnt := g.ecs.Parent(common.EntityId(1))

		if parEnt > -1 {
			err = g.ecs.Detach(common.EntityId(1))
			if err != nil {
				log.Println("error detaching gun: ", err)
			}
		} else {
			err = g.ecs.Attach(common.EntityId(1), g.playerEntity, ecs.Link_Held)
			if err != nil {
				log.Println("error attaching gun to player: ", err)
			}
		}
	}

	pWorldPos, err := g.ecs.WorldPos(g.playerEntity)
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

	if g.ecs.TickIdx%(data.WorldSavePeriodSec*data.TPS) == 0 {
		g.startWorldSave()
	}

	g.ecs.TickIdx++

	return nil
}

func (g *game) logConsistency(when string) {
	if err := g.ecs.CheckConsistency(); err != nil {
		log.Printf("consistency check %s failed:\n%v", when, err)
	}
}

func (g *game) startWorldSave() {
	if g.chunkContainer.IsCommitting() {
		log.Println("skipping world save, the previous one is still being written")
		return
	}

	g.logConsistency("before saving")

	header, err := world.Snapshot(g.playerEntity, g.ecs)
	if err != nil {
		log.Println("error taking world snapshot: ", err)
		return
	}

	if err := g.chunkContainer.StartCommit(header, g.ecs); err != nil {
		log.Println("error starting world save: ", err)
	}
}

func (g *game) saveAndExit() error {
	// Nothing has been simulated yet, so the saved state is still up to date
	if g.loading {
		return ebiten.Termination
	}

	g.logConsistency("before saving")

	header, err := world.Snapshot(g.playerEntity, g.ecs)
	if err != nil {
		log.Println("error taking world snapshot: ", err)
		return ebiten.Termination
	}

	if err := g.chunkContainer.CommitAndWait(header, g.ecs); err != nil {
		log.Println("error saving world: ", err)
	}

	return ebiten.Termination
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	if g.loading {
		ebitenutil.DebugPrint(screen, "Loading...")
		return
	}

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

		pVel, ok := g.ecs.Velocities.Get(g.playerEntity)
		if !ok {
			log.Fatalf("player %d has no velocity to show for debug", g.playerEntity)
		}
		pLocalVelVec := pVel.LocalVector()

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

	g.chunkContainer = tilesystem.NewChunkContainer()
	err := g.chunkContainer.GenerateTileAtlasFromJson("assets/tiles/atlas.json")
	if err != nil {
		log.Fatal("error generating tile atlas from json: ", err)
	}

	if err := content.Register(g.ecs); err != nil {
		log.Fatal("error registering content: ", err)
	}

	g.ecs.TickState = *common.NewTickState()
	g.ecs.SetSimulatedPosCheck(g.chunkContainer.IsActiveWorldPos)

	header, worldExists, err := world.Load(g.ecs)
	if err != nil {
		log.Fatal("error loading world: ", err)
	}

	if worldExists {
		if err := tilesystem.RecoverChunkFiles(header.Generation); err != nil {
			log.Fatal("error recovering chunk files: ", err)
		}
		g.chunkContainer.SetGeneration(header.Generation)
		g.playerEntity = header.PlayerEntity
	} else {
		if err := world.Reset(); err != nil {
			log.Fatal("error resetting world: ", err)
		}
		g.playerEntity, err = templates.SpawnPlayer(g.ecs)
		if err != nil {
			log.Fatal("error spawning player: ", err)
		}
		g.newWorld = true
	}

	g.loading = true

	ebiten.SetWindowSize(1920, 1080)
	ebiten.SetWindowTitle("ebittest")
	ebiten.SetWindowClosingHandled(true)

	if err := ebiten.RunGame(&g); err != nil {
		log.Fatal(err)
	}
}
