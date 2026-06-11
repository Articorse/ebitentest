package main

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/inputsources"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/animationsystem"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/ecs/systems/commonsystems"
	"ebittest/ecs/systems/damagesystem"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/ecs/systems/movementsystem"
	"ebittest/ecs/systems/platformsystem"
	"ebittest/ecs/systems/timersystem"
	"ebittest/ecs/timerfuncs"
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
	g = game{world: ecs.NewWorld()}

	DEBUG_LEVEL                      = 0
	max_vel                          = 0.0
	prev_pos                         = utils.Vec2{}
	max_pos_diff                     = 0.0
	resolvedPhysicsCollisions uint64 = 0

	tm = ecs.TransformManager{}
	pm = ecs.ParentManager{}
	vm = ecs.VelocityManager{}
)

type game struct {
	world        *ecs.World
	playerEntity common.EntityId

	camera       utils.Vec2
	cameraFollow bool

	inputLog map[uint64]map[common.EntityId]ecs.InputState

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
	im := ecs.InputManager{}
	pm := ecs.ParentManager{}

	tickInputs := make(map[common.EntityId]ecs.InputState)
	for _, eid := range g.world.Inputs.GetOrderedEntities() {
		inputSourceFunc, err := im.GetInputSourceFunc(eid, g.world)
		if err != nil {
			log.Printf("error getting input source func for entity %d: %v\n", eid, err)
			continue
		}

		tickInputs[eid] = inputSourceFunc(eid, g.world.TickIdx, g.world)
	}
	g.inputLog[g.world.TickIdx] = tickInputs

	err = inputsystem.HandleInputs(
		g.camera,
		g.world,
		g.inputLog[g.world.TickIdx],
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

		j, err := json.Marshal(g.inputLog)
		if err != nil {
			log.Println("error marshalling replay log: ", err)
		}

		_, err = f.Write(j)
		if err != nil {
			log.Println("error writing replay file: ", err)
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.camera.X -= 10
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.camera.Y -= 10
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.camera.X += 10
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.camera.Y += 10
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.cameraFollow = !g.cameraFollow
	}
	if g.cameraFollow {
		pWorldPos, err := tm.GetWorldPos(g.playerEntity, g.world)
		if err != nil {
			log.Println("error getting player world position for camera follow: ", err)
		}

		g.camera = pWorldPos.Subtract(utils.Vec2{X: float64(data.CameraWidth) / 2, Y: float64(data.CameraHeight) / 2})
	}

	// Replay
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		if !g.recording {
			g.recording = true
			g.recordingStartTick = g.world.TickIdx
			g.recordedInputs = make(map[uint64]ecs.InputState)
		} else {
			g.recording = false
			// TODO: Save to file
		}
	}
	if g.recording {
		relTick := g.world.TickIdx - g.recordingStartTick
		g.recordedInputs[relTick] = g.inputLog[g.world.TickIdx][g.playerEntity]
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		g.replaying = true
		g.replayStartTick = g.world.TickIdx
		// TODO: Load from file
		g.replayInputs = g.recordedInputs

		replaySource := inputsources.NewReplayInputSource(g.replayStartTick, g.replayInputs)

		err := im.SetInputSourceFunc(g.replayEntity, replaySource, g.world)
		if err != nil {
			log.Println("error setting replay input source func: ", err)
		}
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
		parEnt := pm.GetEntity(common.EntityId(1), g.world)

		if parEnt > -1 {
			err = pm.Detach(common.EntityId(1), g.world)
			if err != nil {
				log.Println("error detaching gun: ", err)
			}
		} else {
			err = pm.Attach(common.EntityId(1), g.playerEntity, g.world)
			if err != nil {
				log.Println("error attaching gun to player: ", err)
			}
		}
	}

	pWorldPos, err := tm.GetWorldPos(g.playerEntity, g.world)
	if err != nil {
		log.Println("error getting player world position for debug: ", err)
	} else {
		pos_diff := prev_pos.Subtract(pWorldPos).Length()
		if max_pos_diff < pos_diff {
			max_pos_diff = pos_diff
		}
		prev_pos = pWorldPos
	}

	// END DEBUG

	err = animationsystem.Tick(g.world)
	if err != nil {
		log.Println("animation system tick error: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	err = timersystem.Tick(g.world)
	if err != nil {
		log.Println("timer system tick error: ", err)
	}
	g.world.TickState = *common.NewTickState()

	if err := movementsystem.Tick(g.world); err != nil {
		log.Println("movement system error: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	g.world.TickState.CollisionGrid, err = commonsystems.PopulateSpatialHashGrid(g.world)
	if err != nil {
		log.Println("error during populating spatial hash grid: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	proximateEntities, err := commonsystems.GetSHGProximities(g.world.TickState.CollisionGrid, g.world)
	if err != nil {
		log.Println("error during spatial hash grid proximity checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	pcm := ecs.PhysicsColliderManager{}

	physicsAABBCollisions, err := collisionsystem.GetAABBCollisions(pcm, pcm, proximateEntities, g.world)
	if err != nil {
		log.Println("error during AABB collision checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	physicsCollisions, err := collisionsystem.GetCollisions(pcm, pcm, physicsAABBCollisions, g.world)
	if err != nil {
		log.Println("error during physics collision checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	resolvedPhysicsCollisions, err = collisionsystem.ResolvePhysicsCollisions(physicsCollisions, g.world)
	if err != nil {
		log.Println("error during collision resolution: ", err)
	}

	hurtcm := ecs.HurtboxColliderManager{}
	hitcm := ecs.HitboxColliderManager{}

	damageAABBCollisions, err := collisionsystem.GetMirrorAABBCollisions(hurtcm, hitcm, proximateEntities, g.world)
	if err != nil {
		log.Println("error during AABB damage collision checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	damageCollisions, err := collisionsystem.GetCollisions(hurtcm, hitcm, damageAABBCollisions, g.world)
	if err != nil {
		log.Println("error during damage collision checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	damagesystem.Tick(g.world)

	_, err = damagesystem.DealContactDamage(damageCollisions, g.world)
	if err != nil {
		log.Println("error during dealing contact damage: ", err)
	}

	err = platformsystem.Tick(g.world.TickState.CollisionGrid, g.world)
	if err != nil {
		log.Println("platform system tick error: ", err)
	}

	g.world.TickState.ProximateEntities = proximateEntities
	g.world.TickState.AABBCollisions = physicsAABBCollisions
	g.world.TickState.Collisions = physicsCollisions

	g.world.TickIdx++

	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	if err := drawsystem.DrawFrame(
		screen,
		g.camera,
		g.world.TickState.CollisionGrid,
		g.world,
	); err != nil {
		log.Println("error while drawing frame, removing offending entity: ", err)
		var missingDependencyError *common.ErrorMissingComponentDependency
		if errors.As(err, &missingDependencyError) {
			g.world.RemoveEntity(missingDependencyError.Entity)
		}
		var missingExpectedComponentError *common.ErrorMissingExpectedComponent
		if errors.As(err, &missingExpectedComponentError) {
			g.world.RemoveEntity(missingExpectedComponentError.Entity)
		}
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
		// pcm := ecs.PhysicsColliderManager{}
		hurtcm := ecs.HurtboxColliderManager{}
		hitcm := ecs.HitboxColliderManager{}

		// if err := collisionsystem.DrawColliders(pcm, data.Debug_ColliderColor, data.Debug_ColliderCollidedColor, screen, g.camera, g.world.TickState.Collisions, g.world); err != nil {
		// 	log.Println("error while drawing physics colliders: ", err, "removing entity")
		// 	var invalidComponentsErr *common.ErrorMissingComponentDependency
		// 	if errors.As(err, &invalidComponentsErr) {
		// 		g.world.RemoveEntity(invalidComponentsErr.Entity)
		// 	}
		// }

		if err := collisionsystem.DrawColliders(hurtcm, data.Debug_ColliderColor, data.Debug_ColliderCollidedColor, screen, g.camera, g.world.TickState.Collisions, g.world); err != nil {
			log.Println("error while drawing physics colliders: ", err, "removing entity")
			var invalidComponentsErr *common.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawColliders(hitcm, data.Debug_ColliderColor, data.Debug_ColliderCollidedColor, screen, g.camera, g.world.TickState.Collisions, g.world); err != nil {
			log.Println("error while drawing physics colliders: ", err, "removing entity")
			var invalidComponentsErr *common.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		// if err := collisionsystem.DrawAABBs(pcm, data.Debug_AABBColliderColor, data.Debug_AABBColliderColor, screen, g.camera, g.world.TickState.AABBCollisions, g.world); err != nil {
		// 	log.Println("error while drawing AABBs: ", err, "removing entity")
		// 	var invalidComponentsErr *common.ErrorMissingComponentDependency
		// 	if errors.As(err, &invalidComponentsErr) {
		// 		g.world.RemoveEntity(invalidComponentsErr.Entity)
		// 	}
		// }

		if err := collisionsystem.DrawAABBs(hurtcm, data.Debug_AABBColliderColor, data.Debug_AABBColliderColor, screen, g.camera, g.world.TickState.AABBCollisions, g.world); err != nil {
			log.Println("error while drawing AABBs: ", err, "removing entity")
			var invalidComponentsErr *common.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawAABBs(hitcm, data.Debug_AABBColliderColor, data.Debug_AABBColliderColor, screen, g.camera, g.world.TickState.AABBCollisions, g.world); err != nil {
			log.Println("error while drawing AABBs: ", err, "removing entity")
			var invalidComponentsErr *common.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawCollisions(screen, data.Debug_CollisionVectorColor, g.camera, g.world.TickState.Collisions, g.world); err != nil {
			log.Println("error while drawing collisions: ", err)
		}

		proximateEntitiesCount := 0
		for _, others := range g.world.TickState.ProximateEntities {
			for range others {
				proximateEntitiesCount++
			}
		}

		pLocalVelVec, err := vm.GetLocalVector(g.playerEntity, g.world)
		if err != nil {
			log.Fatalf("error getting player local velocity for debug: %v", err)
		}

		vel := pLocalVelVec.Length()
		if vel > max_vel {
			max_vel = vel
		}

		ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %f\nTick: %d\nVel: %v\nMaxVel: %f\nMaxPosDiff: %v\nSHG Cells: %v\nProximate Pairs: %d\nResolved Collisions: %d", ebiten.ActualFPS(), g.world.TickIdx, pLocalVelVec, max_vel, max_pos_diff, g.world.TickState.CollisionGrid, proximateEntitiesCount, resolvedPhysicsCollisions))
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(data.TPS)

	g.inputLog = make(map[uint64]map[common.EntityId]ecs.InputState)
	g.world.TickState = *common.NewTickState()

	// Player
	pInputConfig := ecs.InputConfig{
		Up:    ebiten.KeyW,
		Down:  ebiten.KeyS,
		Left:  ebiten.KeyA,
		Right: ebiten.KeyD,
		Use:   ebiten.MouseButtonLeft,
	}

	pInpComp := ecs.NewInputComponent(pInputConfig, inputsources.KeyboardInputSource)
	pParComp := ecs.NewParentComponent()
	pTraComp := ecs.NewTransformComponent(utils.Vec2{X: 100, Y: 100}, 1, 0)
	pVelComp := ecs.NewDefaultVelocityComponent()
	pSprComp, err := ecs.NewSpriteComponent("assets/sprites/slime.png", 20, true)
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
	pAniComp, err := ecs.NewAnimationComponent("assets/sprites/slime_ss.png", utils.Vec2{X: 32, Y: 32}, pAniStateFrames)
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

	g.playerEntity = g.world.AddEntity(
		pInpComp,
		pParComp,
		pTraComp,
		pVelComp,
		pSprComp,
		pAniComp,
		pPhyColComp,
		pHpComp,
		pHitboxComp,
	)

	gunParComp := ecs.NewParentComponent()
	gunTraComp := ecs.NewTransformComponent(utils.Vec2{X: 100, Y: 100}, 1, 0)
	gunVelComp := ecs.NewDefaultVelocityComponent()
	gunSprComp, err := ecs.NewSpriteComponent("assets/sprites/gun.png", 20, true)
	if err != nil {
		log.Fatal(err)
	}
	gunAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	gunAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 1000},
	}
	gunAniStateFrames[ecs.Anim_Use] = []ecs.AnimationFrame{
		{FrameIdx: 1, DurationMs: 100},
	}
	gunAniComp, err := ecs.NewAnimationComponent("assets/sprites/gun_ss.png", utils.Vec2{X: 32, Y: 32}, gunAniStateFrames)
	gunInpComp := ecs.NewInputComponent(ecs.InputConfig{}, inputsources.MouseInputSource)

	bulletTraComp := ecs.NewTransformComponent(utils.Vec2{}, 1, 0)
	bulletVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2{X: 5, Y: 0}, 0, 1)
	bulletTimerComp, err := ecs.NewTimerComponent(500, 1, timerfuncs.Selfdestruct)
	if err != nil {
		log.Fatal("error creating bullet timer component: ", err)
	}
	bulletDmgComp := ecs.NewContactDamageComponent(g.playerEntity, 10, true, 5)
	bulletSprComp, err := ecs.NewSpriteComponent("assets/sprites/bullet.png", 20, false)
	if err != nil {
		log.Fatal(err)
	}
	bulletAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	bulletAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 50},
		{FrameIdx: 1, DurationMs: 50},
		{FrameIdx: 2, DurationMs: 50},
		{FrameIdx: 1, DurationMs: 50},
	}
	bulletAniComp, err := ecs.NewAnimationComponent("assets/sprites/bullet_ss.png", utils.Vec2{X: 32, Y: 32}, bulletAniStateFrames)
	bulletHurtboxShape, err := shapes.NewCircleShape(3, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
	}
	bulletHurtboxComp := ecs.NewHurtboxColliderComponent(
		ecs.Layer_FriendlyProjectile,
		ecs.Layer_Enemy|
			ecs.Layer_Terrain,
		bulletHurtboxShape,
	)

	gunSpaComp, err := ecs.NewSpawnerComponent(
		utils.Vec2{X: 13, Y: 0},
		ecs.SpawnerType_Point,
		nil,
		bulletTraComp, bulletSprComp, bulletVelComp, bulletDmgComp, bulletHurtboxComp, bulletAniComp, bulletTimerComp,
	)
	if err != nil {
		log.Fatal("error creating gun spawner component: ", err)
	}

	enemyTraComp := ecs.NewTransformComponent(utils.Vec2{X: 300, Y: 150}, 1, 0)
	enemyVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2{}, data.DefaultAcceleration*0.25, data.DefaultDrag)
	enemyParComp := ecs.NewParentComponent()
	enemySprComp, err := ecs.NewSpriteComponent("assets/sprites/evilslime.png", 20, true)
	if err != nil {
		log.Fatal(err)
	}
	enemyAniStateFrames := make(map[ecs.AnimationState][]ecs.AnimationFrame)
	enemyAniStateFrames[ecs.Anim_Idle] = []ecs.AnimationFrame{
		{FrameIdx: 0, DurationMs: 2000},
		{FrameIdx: 1, DurationMs: 100},
		{FrameIdx: 2, DurationMs: 100},
		{FrameIdx: 3, DurationMs: 100},
		{FrameIdx: 4, DurationMs: 100},
		{FrameIdx: 3, DurationMs: 100},
		{FrameIdx: 2, DurationMs: 100},
		{FrameIdx: 1, DurationMs: 100},
	}
	enemyAniComp, err := ecs.NewAnimationComponent("assets/sprites/evilslime_ss.png", utils.Vec2{X: 32, Y: 32}, enemyAniStateFrames)
	enemyHpComp := ecs.NewHitpointsComponent(20, 100)
	enemyPhyColShape, err := shapes.NewCircleShape(7, utils.Vec2{X: 0, Y: 0})
	enemyPhyColComp := ecs.NewPhysicsColliderComponent(
		ecs.Layer_Enemy,
		ecs.Layer_Player|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Platform|
			ecs.Layer_Enemy|
			ecs.Layer_Terrain,
		ecs.Collider_Mob,
		enemyPhyColShape,
	)
	enemyHitboxColShape, err := shapes.NewCircleShape(5, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
	}
	enemyHitboxComp := ecs.NewHitboxColliderComponent(
		ecs.Layer_Enemy,
		ecs.Layer_Player|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Platform|
			ecs.Layer_Terrain,
		enemyHitboxColShape,
	)
	enemyHurtboxColShape, err := shapes.NewCircleShape(7, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
	}
	enemyHurtboxComp := ecs.NewHurtboxColliderComponent(
		ecs.Layer_Enemy,
		ecs.Layer_Player|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Platform|
			ecs.Layer_Terrain,
		enemyHurtboxColShape,
	)
	enemyCDmgComp := ecs.NewContactDamageComponent(-1, 20, false, 1)
	enemyFollowInput := inputsources.NewFollowInputSource(g.playerEntity)
	enemyInputComp := ecs.NewInputComponent(ecs.InputConfig{}, enemyFollowInput)

	_ = g.world.AddEntity(
		enemyParComp,
		enemyTraComp,
		enemyVelComp,
		enemySprComp,
		enemyAniComp,
		enemyHpComp,
		enemyPhyColComp,
		enemyHitboxComp,
		enemyHurtboxComp,
		enemyCDmgComp,
		enemyInputComp,
	)
	enemySpawnerTimerComp, err := ecs.NewTimerComponent(1000, -1, timerfuncs.Spawn)
	enemySpawnerShape, err := shapes.NewRectangleShape(data.CameraWidth+100, data.CameraHeight+100, utils.Vec2{})
	if err != nil {
		log.Fatal(err)
	}
	enemySpawnerComp, err := ecs.NewSpawnerComponent(
		utils.Vec2{
			X: g.camera.X + (data.CameraWidth / 2),
			Y: g.camera.Y + (data.CameraHeight / 2),
		},
		ecs.SpawnerType_Perimeter,
		enemySpawnerShape,
		enemyParComp,
		enemyTraComp,
		enemyVelComp,
		enemySprComp,
		enemyAniComp,
		enemyHpComp,
		enemyPhyColComp,
		enemyHitboxComp,
		enemyHurtboxComp,
		enemyCDmgComp,
		enemyInputComp,
	)

	g.world.AddEntity(enemySpawnerTimerComp, enemySpawnerComp)
	gun := g.world.AddEntity(
		gunInpComp,
		gunParComp,
		gunTraComp,
		gunVelComp,
		gunSprComp,
		gunAniComp,
		gunSpaComp,
	)

	err = pm.Attach(gun, g.playerEntity, g.world)
	if err != nil {
		log.Fatal("error attaching gun to player: ", err)
	}

	var inputLoop []ecs.InputState

	for range 50 {
		inputLoop = append(inputLoop, ecs.InputState{Analog1X: -1})
	}
	for range 50 {
		inputLoop = append(inputLoop, ecs.InputState{Analog1X: 1})
	}

	loopSource := inputsources.NewLoopInputSource(inputLoop, 0)

	treeInpComp := ecs.NewInputComponent(ecs.InputConfig{}, inputsources.DummyInputSource)
	treeParComp := ecs.NewParentComponent()
	treeTraComp := ecs.NewTransformComponent(utils.Vec2{X: 450, Y: 250}, 1, 0)
	treeVelComp := ecs.NewDefaultVelocityComponent()
	treeSprComp, err := ecs.NewSpriteComponent("assets/sprites/tree.png", 20, true)
	if err != nil {
		log.Fatal(err)
	}

	treeColShape, err := shapes.NewRectangleShape(10, 5, utils.Vec2{X: -1, Y: 9})
	// treeColShape, err := shapes.NewCircleShape(5, utils.Vec2{X: -1, Y: 9})
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

	tree := g.world.AddEntity(
		treeInpComp,
		treeParComp,
		treeTraComp,
		treeVelComp,
		treeSprComp,
		treeColComp,
	)

	g.replayEntity = tree

	platInput := ecs.NewInputComponent(ecs.InputConfig{}, loopSource)

	platParComp := ecs.NewParentComponent()
	platTraComp := ecs.NewTransformComponent(utils.Vec2{X: 250, Y: 100}, 1, 0)
	platVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2{}, 0.3, data.DefaultDrag)

	platSprComp, err := ecs.NewSpriteComponent("assets/sprites/platform.png", 10, true)
	if err != nil {
		log.Fatal(err)
	}

	platColShape, err := shapes.NewRectangleShape(28, 28, utils.Vec2{X: 0, Y: 0})
	platColComp := ecs.NewPlatformColliderComponent(
		ecs.Layer_Platform,
		ecs.Layer_Player|
			ecs.Layer_Enemy|
			ecs.Layer_EnemyProjectile|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Terrain,
		[]shapes.Shape{platColShape},
	)

	_ = g.world.AddEntity(
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
	x := g.world.Rng.IntN(10000)
	y := g.world.Rng.IntN(10000)

	traComp := ecs.NewTransformComponent(utils.Vec2{X: float64(x), Y: float64(y)}, 1, 0)
	sprComp, err := ecs.NewSpriteComponent("assets/sprites/tree.png", 20, true)
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

	_ = g.world.AddEntity(
		traComp,
		velComp,
		sprComp,
		colComp,
	)
}
