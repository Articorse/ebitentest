package main

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/components"
	"ebittest/ecs/components/hitboxes"
	"ebittest/ecs/components/inputsources"
	"ebittest/ecs/ecscommon"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/ecs/systems/commonsystems"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/ecs/systems/movementsystem"
	"ebittest/ecs/systems/platformsystem"
	"ebittest/utils"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"log"
	"maps"
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

	tm = components.TransformManager{}
	pm = components.ParentManager{}
	vm = components.VelocityManager{}
)

type game struct {
	world        *ecs.World
	playerEntity ecscommon.EntityId

	tickIdx   uint64
	tickState ecscommon.TickState

	camera       utils.Vec2
	cameraFollow bool

	inputLog map[uint64]map[ecscommon.EntityId]components.InputState

	recording          bool
	recordingStartTick uint64
	recordedInputs     map[uint64]components.InputState

	replaying       bool
	replayStartTick uint64
	replayInputs    map[uint64]components.InputState
	replayEntity    ecscommon.EntityId
}

func (g *game) Update() error {
	// pCenterX := g.p.x + float64(g.p.img.Bounds().Dx())/2
	// pCenterY := g.p.y + float64(g.p.img.Bounds().Dy())/2
	// dX := float64(mX) - pCenterX
	// dY := float64(mY) - pCenterY
	// r = math.Atan2(dY, dX)
	var err error
	im := components.InputManager{}
	pm := components.ParentManager{}

	tickInputs := make(map[ecscommon.EntityId]components.InputState)
	for eid, _ := range g.world.Inputs {
		inputSourceFunc, err := im.GetInputSourceFunc(eid, g.world.Inputs)
		if err != nil {
			log.Printf("error getting input source func for entity %d: %v\n", eid, err)
			continue
		}

		tickInputs[eid] = inputSourceFunc(eid, g.tickIdx, g.world.Inputs)
	}
	g.inputLog[g.tickIdx] = tickInputs

	err = inputsystem.HandleInputs(g.world.Velocities, g.inputLog[g.tickIdx])
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
			g.recordedInputs = make(map[uint64]components.InputState)
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

	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		maps.DeleteFunc(g.world.Transforms,
			func(k ecscommon.EntityId, _ *components.Transform) bool { return k == g.playerEntity })
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		parEnt := pm.GetEntity(ecscommon.EntityId(1), g.world.Parents)

		if parEnt > -1 {
			err = pm.Detach(ecscommon.EntityId(1), g.world.Transforms, g.world.Parents)
			if err != nil {
				log.Println("error detaching gun: ", err)
			}
		} else {
			err = pm.Attach(ecscommon.EntityId(1), g.playerEntity, g.world.Transforms, g.world.Parents)
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

	g.tickState = *ecscommon.NewTickState()

	if err := movementsystem.TickEarly(g.world.Velocities, g.world.Transforms, g.world.Parents); err != nil {
		log.Println("movement system error: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	g.tickState.CollisionGrid, err = commonsystems.PopulateSpatialHashGrid(g.world.Transforms, g.world.Parents)
	if err != nil {
		log.Println("error during populating spatial hash grid: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	proximateEntities, err := commonsystems.GetSHGProximities(g.tickState.CollisionGrid, g.world.Colliders, g.world.Transforms, g.world.Parents)
	if err != nil {
		log.Println("error during spatial hash grid proximity checking: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	aabbcollisions, err := collisionsystem.GetAABBCollisions(proximateEntities, g.world.Colliders, g.world.Transforms, g.world.Parents)
	if err != nil {
		log.Println("error during AABB collision checking: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	collisions, err := collisionsystem.GetCollisions(aabbcollisions, g.world.Colliders, g.world.Transforms, g.world.Velocities, g.world.Parents)
	if err != nil {
		log.Println("error during collision checking: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	resolvedCollisions, err = collisionsystem.ResolveCollisions(collisions, g.world.Colliders, g.world.Transforms, g.world.Velocities)
	if err != nil {
		log.Println("error during collision resolution: ", err)
	}

	err = platformsystem.Tick(g.tickState.CollisionGrid, g.world.Platforms, g.world.Transforms, g.world.Colliders, g.world.Parents)
	if err != nil {
		log.Println("platform system tick error: ", err)
	}

	err = movementsystem.TickLate(g.world.Transforms)
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
		g.world.Sprites,
		g.world.Transforms,
		g.world.Parents,
	); err != nil {
		log.Println("error while drawing frame, removing offending entity: ", err)
		var missingDependencyError *ecscommon.ErrorMissingComponentDependency
		if errors.As(err, &missingDependencyError) {
			g.world.RemoveEntity(missingDependencyError.Entity)
		}
		var missingExpectedComponentError *ecscommon.ErrorMissingExpectedComponent
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
		if err := collisionsystem.DrawColliders(screen, g.camera, g.world.Colliders, g.world.Transforms, g.tickState.Collisions, g.world.Parents); err != nil {
			log.Println("error while drawing colliders: ", err, "removing entity")
			var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawAABBs(screen, g.camera, g.world.Colliders, g.world.Transforms, g.world.Parents, g.tickState.AABBCollisions); err != nil {
			log.Println("error while drawing AABBs: ", err, "removing entity")
			var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawCollisions(screen, g.camera, g.tickState.Collisions, g.world.Transforms, g.world.Parents); err != nil {
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
	g.inputLog = make(map[uint64]map[ecscommon.EntityId]components.InputState)
	g.tickState = *ecscommon.NewTickState()

	// Player
	g.playerEntity = g.world.AddEntity()

	pInputConfig := components.InputConfig{
		Up:    ebiten.KeyW,
		Down:  ebiten.KeyS,
		Left:  ebiten.KeyA,
		Right: ebiten.KeyD,
		Use:   ebiten.MouseButtonLeft,
	}

	pInpComp := components.NewInputComponent(pInputConfig, inputsources.KeyboardMouseInputSource)
	pParComp := components.NewParentComponent()
	pTraComp := components.NewTransformComponent(utils.Vec2{X: 100, Y: 100}, 1, 0)
	pVelComp := components.NewVelocityComponent()
	pSprComp, err := components.NewSpriteComponent("assets/sprites/slime.png", 20)
	if err != nil {
		log.Fatal(err)
	}

	pHitbox, err := hitboxes.NewCircleHitbox(5, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
	}

	pColComp := components.NewColliderComponent(
		components.Collider_Mob,
		[]hitboxes.Hitbox{pHitbox},
		components.Layer_Player,
		components.Layer_EnemyProjectile|
			components.Layer_Terrain|
			components.Layer_Platform,
	)

	g.world.Inputs[g.playerEntity] = pInpComp
	g.world.Parents[g.playerEntity] = pParComp
	g.world.Transforms[g.playerEntity] = pTraComp
	g.world.Velocities[g.playerEntity] = pVelComp
	g.world.Sprites[g.playerEntity] = pSprComp
	g.world.Colliders[g.playerEntity] = pColComp

	gun := g.world.AddEntity()
	gunParComp := components.NewParentComponent()

	gunTraComp := components.NewTransformComponent(utils.Vec2{X: 100, Y: 100}, 1, 0)
	gunVelComp := components.NewVelocityComponent()

	gunSprComp, err := components.NewSpriteComponent("assets/sprites/gun.png", 20)
	if err != nil {
		log.Fatal(err)
	}

	g.world.Parents[gun] = gunParComp
	g.world.Transforms[gun] = gunTraComp
	g.world.Velocities[gun] = gunVelComp
	g.world.Sprites[gun] = gunSprComp

	err = pm.Attach(gun, g.playerEntity, g.world.Transforms, g.world.Parents)
	if err != nil {
		log.Fatal("error attaching gun to player: ", err)
	}

	e := g.world.AddEntity()
	g.replayEntity = e

	var inputLoop []components.InputState

	for range 50 {
		inputLoop = append(inputLoop, components.InputState{Left: true})
	}
	for range 50 {
		inputLoop = append(inputLoop, components.InputState{Right: true})
	}

	loopSource := inputsources.NewLoopInputSource(inputLoop, 0)

	eInpComp := components.NewInputComponent(components.InputConfig{}, inputsources.DummyInputSource)
	g.world.Inputs[e] = eInpComp

	eParComp := components.NewParentComponent()
	eTraComp := components.NewTransformComponent(utils.Vec2{X: 450, Y: 250}, 1, 0)
	eVelComp := components.NewVelocityComponent()
	eSprComp, err := components.NewSpriteComponent("assets/sprites/tree.png", 20)
	if err != nil {
		log.Fatal(err)
	}

	eHitbox, err := hitboxes.NewRectangleHitbox(10, 5, utils.Vec2{X: -1, Y: 9})
	eColComp := components.NewColliderComponent(
		components.Collider_Static,
		[]hitboxes.Hitbox{eHitbox},
		components.Layer_Terrain,
		components.Layer_Player|
			components.Layer_Enemy|
			components.Layer_EnemyProjectile|
			components.Layer_FriendlyProjectile|
			components.Layer_Platform,
	)

	g.world.Parents[e] = eParComp
	g.world.Transforms[e] = eTraComp
	g.world.Velocities[e] = eVelComp
	g.world.Sprites[e] = eSprComp
	g.world.Colliders[e] = eColComp

	plat := g.world.AddEntity()

	platInput := components.NewInputComponent(components.InputConfig{}, loopSource)
	g.world.Inputs[plat] = platInput

	platParComp := components.NewParentComponent()
	platTraComp := components.NewTransformComponent(utils.Vec2{X: 250, Y: 100}, 1, 0)
	platVelComp := components.NewVelocityComponent()

	platSprComp, err := components.NewSpriteComponent("assets/sprites/platform.png", 10)
	if err != nil {
		log.Fatal(err)
	}

	platHitbox, err := hitboxes.NewRectangleHitbox(28, 28, utils.Vec2{X: 0, Y: 0})
	platColComp := components.NewColliderComponent(
		components.Collider_Trigger,
		[]hitboxes.Hitbox{platHitbox},
		components.Layer_Platform,
		components.Layer_Player|
			components.Layer_Enemy|
			components.Layer_Terrain,
	)

	platPlaComp := components.NewPlatformComponent()

	g.world.Parents[plat] = platParComp
	g.world.Transforms[plat] = platTraComp
	g.world.Velocities[plat] = platVelComp
	g.world.Sprites[plat] = platSprComp
	g.world.Colliders[plat] = platColComp
	g.world.Platforms[plat] = platPlaComp

	err = vm.SetAcceleration(plat, 0.3, g.world.Velocities)
	if err != nil {
		log.Fatal("error setting platform acceleration: ", err)
	}

	// err = pm.Attach(g.playerEntity, plat, g.world.Transforms, g.world.Parents)
	// if err != nil {
	// 	log.Fatal("error attaching player to platform: ", err)
	// }

	for _ = range 500 {
		g.AddRandomEntity()
	}

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

	traComp := components.NewTransformComponent(utils.Vec2{X: float64(x), Y: float64(y)}, 1, 0)
	sprComp, err := components.NewSpriteComponent("assets/sprites/tree.png", 20)
	if err != nil {
		log.Fatal(err)
	}

	velComp := components.NewVelocityComponent()
	hitbox, err := hitboxes.NewRectangleHitbox(10, 5, utils.Vec2{X: -1, Y: 9})
	colComp := components.NewColliderComponent(
		components.Collider_Static,
		[]hitboxes.Hitbox{hitbox},
		components.Layer_Terrain,
		components.Layer_Player|
			components.Layer_Enemy|
			components.Layer_EnemyProjectile|
			components.Layer_FriendlyProjectile,
	)

	g.world.Transforms[e] = traComp
	g.world.Sprites[e] = sprComp
	g.world.Velocities[e] = velComp
	g.world.Colliders[e] = colComp
}
