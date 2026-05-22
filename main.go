package main

import (
	"ebittest/ecs"
	"ebittest/ecs/components"
	"ebittest/ecs/components/hitboxes"
	"ebittest/ecs/ecscommon"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/ecs/systems/commonsystems"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/ecs/systems/movementsystem"
	"ebittest/utils"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"log"
	"maps"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	g = game{world: ecs.NewWorld()}

	height = 360
	width  = 640

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

	inputLog map[uint64]map[ecscommon.EntityId]ecscommon.InputState

	recording          bool
	recordingStartTick uint64
	recordedInputs     map[uint64]ecscommon.InputState

	replaying       bool
	replayStartTick uint64
	replayInputs    map[uint64]ecscommon.InputState
	replayEntity    ecscommon.EntityId
}

func KeyboardMouseInputSource(entityId ecscommon.EntityId, tick uint64) ecscommon.InputState {
	is := ecscommon.InputState{}

	if ebiten.IsKeyPressed(g.world.Inputs[entityId].Left) {
		is.Left = true
	}
	if ebiten.IsKeyPressed(g.world.Inputs[entityId].Right) {
		is.Right = true
	}
	if ebiten.IsKeyPressed(g.world.Inputs[entityId].Up) {
		is.Up = true
	}
	if ebiten.IsKeyPressed(g.world.Inputs[entityId].Down) {
		is.Down = true
	}

	mX, mY := ebiten.CursorPosition()
	is.MousePos = utils.Vec2{X: float64(mX), Y: float64(mY)}
	if inpututil.IsMouseButtonJustPressed(g.world.Inputs[entityId].Use) {
		is.Use = true
		fmt.Println("use key just pressed")
	}

	return is
}

func DemoInputSource(log map[uint64]map[ecscommon.EntityId]ecscommon.InputState) ecscommon.InputSourceFunc {
	return func(entityId ecscommon.EntityId, tick uint64) ecscommon.InputState {
		return log[tick][entityId]
	}
}

func DummyInputSource(entityId ecscommon.EntityId, tick uint64) ecscommon.InputState {
	return ecscommon.InputState{}
}

func (g *game) Update() error {
	// pCenterX := g.p.x + float64(g.p.img.Bounds().Dx())/2
	// pCenterY := g.p.y + float64(g.p.img.Bounds().Dy())/2
	// dX := float64(mX) - pCenterX
	// dY := float64(mY) - pCenterY
	// r = math.Atan2(dY, dX)
	var err error

	tickInputs := make(map[ecscommon.EntityId]ecscommon.InputState)
	for eid, inputComp := range g.world.Inputs {
		tickInputs[eid] = inputComp.InputSourceFunc(eid, g.tickIdx)
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

		g.camera = pWorldPos.Subtract(utils.Vec2{X: float64(width) / 2, Y: float64(height) / 2})
	}

	// Replay
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		if !g.recording {
			g.recording = true
			g.recordingStartTick = g.tickIdx
			g.recordedInputs = make(map[uint64]ecscommon.InputState)
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
		g.world.Inputs[g.replayEntity].InputSourceFunc = func(entityId ecscommon.EntityId, tick uint64) ecscommon.InputState {
			relTick := tick - g.replayStartTick
			relTickInput, ok := g.replayInputs[relTick]
			if !ok {
				g.world.Inputs[g.replayEntity].InputSourceFunc = DummyInputSource
				return ecscommon.InputState{}
			}
			return relTickInput
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

	gParComp, ok := g.world.Parents[ecscommon.EntityId(1)]
	if !ok {
		log.Println("player has no parent component")
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		if gParComp.Entity > -1 {
			err = tm.Detach(ecscommon.EntityId(1), g.world.Transforms, g.world.Parents)
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

	collisions, err := collisionsystem.GetCollisions(aabbcollisions, g.world.Colliders, g.world.Transforms, g.world.Parents)
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
	return width, height
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
			log.Fatalf("error getting player local velocity for debug: ", err)
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

	g.inputLog = make(map[uint64]map[ecscommon.EntityId]ecscommon.InputState)
	g.tickState = *ecscommon.NewTickState()

	g.playerEntity = g.world.AddEntity()

	g.world.Inputs[g.playerEntity] = &components.Input{
		Up:              ebiten.KeyW,
		Down:            ebiten.KeyS,
		Left:            ebiten.KeyA,
		Right:           ebiten.KeyD,
		Use:             ebiten.MouseButtonLeft,
		InputSourceFunc: KeyboardMouseInputSource,
	}

	pParComp := components.NewParentComponent()
	pChiComp := components.NewChildrenComponent()
	pTraComp := components.NewTransformComponent(utils.Vec2{X: 100, Y: 100}, 1, 0)
	pVelComp := components.NewVelocityComponent()
	pSprComp, err := components.NewSpriteComponent("assets/sprites/slime.png")
	if err != nil {
		log.Fatal(err)
	}

	pHitbox, err := hitboxes.NewCircleHitbox(5, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
	}

	pColComp := components.NewColliderComponent(components.Mob, []hitboxes.Hitbox{pHitbox})

	g.world.Parents[g.playerEntity] = pParComp
	g.world.Children[g.playerEntity] = pChiComp
	g.world.Transforms[g.playerEntity] = pTraComp
	g.world.Velocities[g.playerEntity] = pVelComp
	g.world.Sprites[g.playerEntity] = pSprComp
	g.world.Colliders[g.playerEntity] = pColComp

	gun := g.world.AddEntity()
	gunParComp := components.NewParentComponent()

	gunChiComp := components.NewChildrenComponent()
	gunTraComp := components.NewTransformComponent(utils.Vec2{X: 100, Y: 100}, 1, 0)
	gunVelComp := components.NewVelocityComponent()

	gunSprComp, err := components.NewSpriteComponent("assets/sprites/gun.png")
	if err != nil {
		log.Fatal(err)
	}

	gunHitbox, err := hitboxes.NewCircleHitbox(5, utils.Vec2{X: 0, Y: 0})
	if err != nil {
		log.Fatal(err)
	}

	gunColComp := components.NewColliderComponent(components.Mob, []hitboxes.Hitbox{gunHitbox})

	g.world.Parents[gun] = gunParComp
	g.world.Children[gun] = gunChiComp
	g.world.Transforms[gun] = gunTraComp
	g.world.Velocities[gun] = gunVelComp
	g.world.Sprites[gun] = gunSprComp
	g.world.Colliders[gun] = gunColComp

	err = pm.Attach(gun, g.playerEntity, g.world.Transforms, g.world.Parents)
	if err != nil {
		log.Fatal("error attaching gun to player: ", err)
	}

	e := g.world.AddEntity()
	g.replayEntity = e

	g.world.Inputs[e] = &components.Input{
		Up:              ebiten.KeyF24,
		Down:            ebiten.KeyF24,
		Left:            ebiten.KeyF24,
		Right:           ebiten.KeyF24,
		InputSourceFunc: DummyInputSource,
	}

	eParComp := components.NewParentComponent()
	eChiComp := components.NewChildrenComponent()
	eTraComp := components.NewTransformComponent(utils.Vec2{X: 450, Y: 250}, 1, 0)
	eVelComp := components.NewVelocityComponent()
	eSprComp, err := components.NewSpriteComponent("assets/sprites/tree.png")
	if err != nil {
		log.Fatal(err)
	}

	eHitbox, err := hitboxes.NewRectangleHitbox(10, 5, utils.Vec2{X: -1, Y: 9})
	eColComp := components.NewColliderComponent(components.Static, []hitboxes.Hitbox{eHitbox})

	g.world.Parents[e] = eParComp
	g.world.Children[e] = eChiComp
	g.world.Transforms[e] = eTraComp
	g.world.Velocities[e] = eVelComp
	g.world.Sprites[e] = eSprComp
	g.world.Colliders[e] = eColComp

	ebiten.SetWindowSize(1920, 1080)
	ebiten.SetWindowTitle("ebittest")

	if err := ebiten.RunGame(&g); err != nil {
		log.Fatal(err)
	}
}
