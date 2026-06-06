package main

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/collidershapes"
	"ebittest/ecs/common"
	"ebittest/ecs/inputsources"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/ecs/systems/commonsystems"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/ecs/systems/movementsystem"
	"ebittest/ecs/systems/platformsystem"
	"ebittest/ecs/systems/timersystem"
	"ebittest/utils"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	g = game{world: ecs.NewWorld()}

	DEBUG_LEVEL               = 0
	max_vel                   = 0.0
	prev_pos                  = utils.Vec2{}
	max_pos_diff              = 0.0
	resolvedCollisions uint64 = 0

	tm = ecs.TransformManager{}
	pm = ecs.ParentManager{}
	vm = ecs.VelocityManager{}
)

type game struct {
	world        *ecs.World
	playerEntity common.EntityId

	tickIdx   uint64
	tickState common.TickState

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
	for eid, _ := range g.world.Inputs {
		inputSourceFunc, err := im.GetInputSourceFunc(eid, g.world.Inputs)
		if err != nil {
			log.Printf("error getting input source func for entity %d: %v\n", eid, err)
			continue
		}

		tickInputs[eid] = inputSourceFunc(eid, g.tickIdx, g.world)
	}
	g.inputLog[g.tickIdx] = tickInputs

	err = inputsystem.HandleInputs(
		g.camera,
		g.world,
		g.inputLog[g.tickIdx],
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
		pWorldPos, err := tm.GetWorldPos(g.playerEntity, g.world.Transforms, g.world.Parents)
		if err != nil {
			log.Println("error getting player world position for camera follow: ", err)
		}

		g.camera = pWorldPos.Subtract(utils.Vec2{X: float64(data.CameraWidth) / 2, Y: float64(data.CameraHeight) / 2})
	}

	// Replay
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		if !g.recording {
			g.recording = true
			g.recordingStartTick = g.tickIdx
			g.recordedInputs = make(map[uint64]ecs.InputState)
		} else {
			g.recording = false
			// TODO: Save to file
		}
	}
	if g.recording {
		relTick := g.tickIdx - g.recordingStartTick
		g.recordedInputs[relTick] = g.inputLog[g.tickIdx][g.playerEntity]
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		g.replaying = true
		g.replayStartTick = g.tickIdx
		// TODO: Load from file
		g.replayInputs = g.recordedInputs

		replaySource := inputsources.NewReplayInputSource(g.replayStartTick, g.replayInputs)

		err := im.SetInputSourceFunc(g.replayEntity, replaySource, g.world.Inputs)
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
		parEnt := pm.GetEntity(common.EntityId(1), g.world.Parents)

		if parEnt > -1 {
			err = pm.Detach(common.EntityId(1), g.world.Transforms, g.world.Parents)
			if err != nil {
				log.Println("error detaching gun: ", err)
			}
		} else {
			err = pm.Attach(common.EntityId(1), g.playerEntity, g.world.Transforms, g.world.Parents)
			if err != nil {
				log.Println("error attaching gun to player: ", err)
			}
		}
	}

	pWorldPos, err := tm.GetWorldPos(g.playerEntity, g.world.Transforms, g.world.Parents)
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

	pcm := ecs.PhysicsColliderManager{}

	err = timersystem.Tick(g.world)
	if err != nil {
		log.Println("timer system tick error: ", err)
	}
	g.tickState = *common.NewTickState()

	if err := movementsystem.TickEarly(g.world); err != nil {
		log.Println("movement system error: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	g.tickState.CollisionGrid, err = commonsystems.PopulateSpatialHashGrid(g.world)
	if err != nil {
		log.Println("error during populating spatial hash grid: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	proximateEntities, err := commonsystems.GetSHGProximities(g.tickState.CollisionGrid, g.world)
	if err != nil {
		log.Println("error during spatial hash grid proximity checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	aabbcollisions, err := collisionsystem.GetAABBCollisions(pcm, pcm, proximateEntities, g.world)
	if err != nil {
		log.Println("error during AABB collision checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	collisions, err := collisionsystem.GetCollisions(pcm, pcm, aabbcollisions, g.world)
	if err != nil {
		log.Println("error during physics collision checking: ", err, "removing entity")
		var invalidComponentsErr *common.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	resolvedCollisions, err = collisionsystem.ResolvePhysicsCollisions(collisions, g.world)
	if err != nil {
		log.Println("error during collision resolution: ", err)
	}

	// _, err = damagesystem.DealContactDamage(triggerCollisions, g.world)
	// if err != nil {
	// 	log.Println("error during dealing contact damage: ", err)
	// }

	err = platformsystem.Tick(g.tickState.CollisionGrid, g.world)
	if err != nil {
		log.Println("platform system tick error: ", err)
	}

	err = movementsystem.TickLate(g.world)
	if err != nil {
		log.Println("movement system late tick error: ", err)
	}

	g.tickState.ProximateEntities = proximateEntities
	g.tickState.AABBCollisions = aabbcollisions
	g.tickState.Collisions = collisions

	g.tickIdx++

	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	if err := drawsystem.DrawFrame(
		screen,
		g.camera,
		g.tickState.CollisionGrid,
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
		pcm := ecs.PhysicsColliderManager{}

		if err := collisionsystem.DrawColliders(pcm, screen, g.camera, g.tickState.Collisions, g.world); err != nil {
			log.Println("error while drawing physics colliders: ", err, "removing entity")
			var invalidComponentsErr *common.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawAABBs(pcm, screen, g.camera, g.tickState.AABBCollisions, g.world); err != nil {
			log.Println("error while drawing AABBs: ", err, "removing entity")
			var invalidComponentsErr *common.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawCollisions(screen, g.camera, g.tickState.Collisions, g.world); err != nil {
			log.Println("error while drawing collisions: ", err)
		}

		proximateEntitiesCount := 0
		for _, others := range g.tickState.ProximateEntities {
			for range others {
				proximateEntitiesCount++
			}
		}

		pLocalVelVec, err := vm.GetLocalVector(g.playerEntity, g.world.Velocities)
		if err != nil {
			log.Fatalf("error getting player local velocity for debug: %v", err)
		}

		vel := pLocalVelVec.Length()
		if vel > max_vel {
			max_vel = vel
		}

		ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %f\nTick: %d\nVel: %v\nMaxVel: %f\nMaxPosDiff: %v\nSHG Cells: %v\nProximate Pairs: %d\nResolved Collisions: %d", ebiten.ActualFPS(), g.tickIdx, pLocalVelVec, max_vel, max_pos_diff, g.tickState.CollisionGrid, proximateEntitiesCount, resolvedCollisions))
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(data.TPS)

	g.inputLog = make(map[uint64]map[common.EntityId]ecs.InputState)
	g.tickState = *common.NewTickState()

	// Player
	g.playerEntity = g.world.AddEntity()

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
	pVelComp := ecs.NewVelocityComponent()
	pSprComp, err := ecs.NewSpriteComponent("assets/sprites/slime.png", 20)
	if err != nil {
		log.Fatal(err)
	}

	pColShape, err := collidershapes.NewCircleShape(5, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
	}

	pColComp := ecs.NewPhysicsColliderComponent(
		ecs.Collider_Mob,
		[]collidershapes.Shape{pColShape},
	)

	pColLayerComp := ecs.NewCollisionLayersComponent(
		ecs.Layer_Player,
		ecs.Layer_EnemyProjectile|
			ecs.Layer_Terrain|
			ecs.Layer_Platform,
	)

	g.world.Inputs[g.playerEntity] = pInpComp
	g.world.Parents[g.playerEntity] = pParComp
	g.world.Transforms[g.playerEntity] = pTraComp
	g.world.Velocities[g.playerEntity] = pVelComp
	g.world.Sprites[g.playerEntity] = pSprComp
	g.world.PhysicsColliders[g.playerEntity] = pColComp
	g.world.CollisionLayers[g.playerEntity] = pColLayerComp

	gun := g.world.AddEntity()
	gunParComp := ecs.NewParentComponent()
	gunTraComp := ecs.NewTransformComponent(utils.Vec2{X: 100, Y: 100}, 1, 0)
	gunVelComp := ecs.NewVelocityComponent()

	gunSprComp, err := ecs.NewSpriteComponent("assets/sprites/gun.png", 20)
	if err != nil {
		log.Fatal(err)
	}

	gunInpComp := ecs.NewInputComponent(ecs.InputConfig{}, inputsources.MouseInputSource)

	bulletTraComp := ecs.NewTransformComponent(utils.Vec2{}, 1, 0)
	bulletVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2{X: 5, Y: 0}, 0, 1)
	bulletTLComp := ecs.NewTimedLifeComponent(500)
	bulletDmgComp := ecs.NewContactDamageComponent(g.playerEntity, []int64{5}, 10)
	bulletSprComp, err := ecs.NewSpriteComponent("assets/sprites/bullet.png", 20)
	if err != nil {
		log.Fatal(err)
	}

	gunSpaComp := ecs.NewSpawnerComponent(
		utils.Vec2{X: 13, Y: 0},
		bulletTraComp, bulletSprComp, bulletVelComp, bulletTLComp, bulletDmgComp,
	)

	g.world.Inputs[gun] = gunInpComp
	g.world.Parents[gun] = gunParComp
	g.world.Transforms[gun] = gunTraComp
	g.world.Velocities[gun] = gunVelComp
	g.world.Sprites[gun] = gunSprComp
	g.world.Spawners[gun] = gunSpaComp

	err = pm.Attach(gun, g.playerEntity, g.world.Transforms, g.world.Parents)
	if err != nil {
		log.Fatal("error attaching gun to player: ", err)
	}

	var inputLoop []ecs.InputState

	for range 50 {
		inputLoop = append(inputLoop, ecs.InputState{Left: true})
	}
	for range 50 {
		inputLoop = append(inputLoop, ecs.InputState{Right: true})
	}

	loopSource := inputsources.NewLoopInputSource(inputLoop, 0)

	tree := g.world.AddEntity()
	g.replayEntity = tree

	treeInpComp := ecs.NewInputComponent(ecs.InputConfig{}, inputsources.DummyInputSource)
	g.world.Inputs[tree] = treeInpComp

	treeParComp := ecs.NewParentComponent()
	treeTraComp := ecs.NewTransformComponent(utils.Vec2{X: 450, Y: 250}, 1, 0)
	treeVelComp := ecs.NewVelocityComponent()
	treeSprComp, err := ecs.NewSpriteComponent("assets/sprites/tree.png", 20)
	if err != nil {
		log.Fatal(err)
	}

	treeColShape, err := collidershapes.NewRectangleShape(10, 5, utils.Vec2{X: -1, Y: 9})
	treeColComp := ecs.NewPhysicsColliderComponent(
		ecs.Collider_Static,
		[]collidershapes.Shape{treeColShape},
	)

	treeColLayerComp := ecs.NewCollisionLayersComponent(
		ecs.Layer_Terrain,
		ecs.Layer_Player|
			ecs.Layer_Enemy|
			ecs.Layer_EnemyProjectile|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Platform,
	)

	g.world.Parents[tree] = treeParComp
	g.world.Transforms[tree] = treeTraComp
	g.world.Velocities[tree] = treeVelComp
	g.world.Sprites[tree] = treeSprComp
	g.world.PhysicsColliders[tree] = treeColComp
	g.world.CollisionLayers[tree] = treeColLayerComp

	enemy := g.world.AddEntity()

	enemyTraComp := ecs.NewTransformComponent(utils.Vec2{X: 300, Y: 150}, 1, 0)
	enemyVelComp := ecs.NewVelocityComponent()
	enemyParComp := ecs.NewParentComponent()
	enemySprComp, err := ecs.NewSpriteComponent("assets/sprites/evilslime.png", 20)
	if err != nil {
		log.Fatal(err)
	}
	enemyHpComp := ecs.NewHitpointsComponent(20)
	enemyColShape, err := collidershapes.NewCircleShape(5, utils.Vec2{X: 0, Y: 0})
	enemyColComp := ecs.NewPhysicsColliderComponent(
		ecs.Collider_Mob,
		[]collidershapes.Shape{enemyColShape},
	)

	enemyColLayerComp := ecs.NewCollisionLayersComponent(
		ecs.Layer_Enemy,
		ecs.Layer_Player|
			ecs.Layer_Enemy|
			ecs.Layer_EnemyProjectile|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Platform,
	)

	enemyInput := ecs.NewInputComponent(ecs.InputConfig{}, inputsources.DummyInputSource)
	g.world.Inputs[enemy] = enemyInput

	g.world.Parents[enemy] = enemyParComp
	g.world.Transforms[enemy] = enemyTraComp
	g.world.Velocities[enemy] = enemyVelComp
	g.world.Sprites[enemy] = enemySprComp
	g.world.Hitpoints[enemy] = enemyHpComp
	g.world.PhysicsColliders[enemy] = enemyColComp
	g.world.CollisionLayers[enemy] = enemyColLayerComp

	plat := g.world.AddEntity()

	platInput := ecs.NewInputComponent(ecs.InputConfig{}, loopSource)
	g.world.Inputs[plat] = platInput

	platParComp := ecs.NewParentComponent()
	platTraComp := ecs.NewTransformComponent(utils.Vec2{X: 250, Y: 100}, 1, 0)
	platVelComp := ecs.NewVelocityComponent()

	platSprComp, err := ecs.NewSpriteComponent("assets/sprites/platform.png", 10)
	if err != nil {
		log.Fatal(err)
	}

	platColShape, err := collidershapes.NewRectangleShape(28, 28, utils.Vec2{X: 0, Y: 0})
	platColComp := ecs.NewPlatformColliderComponent(
		[]collidershapes.Shape{platColShape},
	)

	platColLayerComp := ecs.NewCollisionLayersComponent(
		ecs.Layer_Platform,
		ecs.Layer_Player|
			ecs.Layer_Enemy|
			ecs.Layer_EnemyProjectile|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Terrain,
	)

	g.world.Parents[plat] = platParComp
	g.world.Transforms[plat] = platTraComp
	g.world.Velocities[plat] = platVelComp
	g.world.Sprites[plat] = platSprComp
	g.world.PlatformColliders[plat] = platColComp
	g.world.CollisionLayers[plat] = platColLayerComp

	err = vm.SetAcceleration(plat, 0.3, g.world.Velocities)
	if err != nil {
		log.Fatal("error setting platform acceleration: ", err)
	}

	// err = pm.Attach(g.playerEntity, plat, g.world.Transforms, g.world.Parents)
	// if err != nil {
	// 	log.Fatal("error attaching player to platform: ", err)
	// }

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
	e := g.world.AddEntity()

	x := rand.Intn(10000)
	y := rand.Intn(10000)

	traComp := ecs.NewTransformComponent(utils.Vec2{X: float64(x), Y: float64(y)}, 1, 0)
	sprComp, err := ecs.NewSpriteComponent("assets/sprites/tree.png", 20)
	if err != nil {
		log.Fatal(err)
	}

	velComp := ecs.NewVelocityComponent()
	shape, err := collidershapes.NewRectangleShape(10, 5, utils.Vec2{X: -1, Y: 9})
	colComp := ecs.NewPhysicsColliderComponent(
		ecs.Collider_Static,
		[]collidershapes.Shape{shape},
	)

	colLayerComp := ecs.NewCollisionLayersComponent(
		ecs.Layer_Terrain,
		ecs.Layer_Player|
			ecs.Layer_Enemy|
			ecs.Layer_EnemyProjectile|
			ecs.Layer_FriendlyProjectile|
			ecs.Layer_Platform,
	)

	g.world.Transforms[e] = traComp
	g.world.Sprites[e] = sprComp
	g.world.Velocities[e] = velComp
	g.world.PhysicsColliders[e] = colComp
	g.world.CollisionLayers[e] = colLayerComp
}
