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

	// replay = make(map[uint64]map[ecscommon.PlayerId]ecscommon.InputState)

	DEBUG_LEVEL               = 0
	max_vel                   = 0.0
	prev_pos                  = utils.Vec2{}
	max_pos_diff              = 0.0
	resolvedCollisions uint64 = 0
)

type game struct {
	world        *ecs.World
	tickIdx      uint64
	tickState    ecscommon.TickState
	camera       utils.Vec2
	cameraFollow bool
	inputLog     map[uint64]map[ecscommon.PlayerId]ecscommon.InputState
}

func LocalInputSource(playerId ecscommon.PlayerId, tick uint64) ecscommon.InputState {
	is := ecscommon.InputState{}
	if ebiten.IsKeyPressed(g.world.InputConfigs[playerId].Left) {
		is.Left = true
	}
	if ebiten.IsKeyPressed(g.world.InputConfigs[playerId].Right) {
		is.Right = true
	}
	if ebiten.IsKeyPressed(g.world.InputConfigs[playerId].Up) {
		is.Up = true
	}
	if ebiten.IsKeyPressed(g.world.InputConfigs[playerId].Down) {
		is.Down = true
	}
	return is
}

func ReplayInputSource(log map[uint64]map[ecscommon.PlayerId]ecscommon.InputState) ecscommon.InputSourceFunc {
	return func(playerId ecscommon.PlayerId, tick uint64) ecscommon.InputState {
		return log[tick][playerId]
	}
}

func (g *game) Update() error {
	// mX, mY := ebiten.CursorPosition()
	// pCenterX := g.p.x + float64(g.p.img.Bounds().Dx())/2
	// pCenterY := g.p.y + float64(g.p.img.Bounds().Dy())/2
	// dX := float64(mX) - pCenterX
	// dY := float64(mY) - pCenterY
	// r = math.Atan2(dY, dX)
	var err error

	if len(g.world.PlayerEntities) == 0 {
		log.Fatalf("no player entity found")
	}

	// g.inputLog[g.tickIdx] = inputsystem.GetTickInputs(g.world.Players, g.tickIdx, ReplayInputSource(replay))
	// err = inputsystem.HandleInputs(g.world, g.inputLog[g.tickIdx])
	// if err != nil {
	// 	log.Printf("error during handling inputs: %v", err)
	// }

	g.inputLog[g.tickIdx] = inputsystem.GetTickInputs(g.world.InputConfigs, g.tickIdx, LocalInputSource)
	err = inputsystem.HandleInputs(g.world, g.inputLog[g.tickIdx])
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

	pE, ok := g.world.PlayerEntities["player 1"]
	if !ok {
		log.Fatalf("player 1 not found")
	}

	pTraComp, ok := g.world.Transforms[pE]
	if !ok {
		log.Fatalf("player entity does not have a transform component")
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.cameraFollow = !g.cameraFollow
	}
	if g.cameraFollow {
		g.camera = pTraComp.GetPos().Subtract(utils.Vec2{X: float64(width) / 2, Y: float64(height) / 2})
	}

	// DEBUG: For testing purposes only
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		maps.DeleteFunc(g.world.Transforms,
			func(k ecscommon.EntityId, _ *components.Transform) bool { return k == pE })
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		DEBUG_LEVEL++
		if DEBUG_LEVEL > 2 {
			DEBUG_LEVEL = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		max_vel = 0
		max_pos_diff = 0
	}

	pos_diff := prev_pos.Subtract(g.world.Transforms[ecscommon.EntityId(0)].GetPos()).Length()
	if max_pos_diff < pos_diff {
		max_pos_diff = pos_diff
	}
	prev_pos = g.world.Transforms[ecscommon.EntityId(0)].GetPos()
	// END DEBUG

	g.tickState = *ecscommon.NewTickState()

	if err := movementsystem.TickEarly(g.world.Velocities, g.world.Transforms); err != nil {
		log.Println("movement system error: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	g.tickState.CollisionGrid, err = commonsystems.PopulateSpatialHashGrid(g.world.Transforms)
	if err != nil {
		log.Println("error during populating spatial hash grid: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	proximateEntities, err := commonsystems.GetSHGProximities(g.tickState.CollisionGrid, g.world.Colliders, g.world.Transforms)
	if err != nil {
		log.Println("error during spatial hash grid proximity checking: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	aabbcollisions, err := collisionsystem.GetAABBCollisions(proximateEntities, g.world.Colliders, g.world.Transforms)
	if err != nil {
		log.Println("error during AABB collision checking: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	collisions, err := collisionsystem.GetCollisions(aabbcollisions, g.world.Colliders, g.world.Transforms)
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
		if err := collisionsystem.DrawColliders(screen, g.camera, g.world.Colliders, g.world.Transforms, g.tickState.Collisions); err != nil {
			log.Println("error while drawing colliders: ", err, "removing entity")
			var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawAABBs(screen, g.camera, g.world.Colliders, g.world.Transforms, g.tickState.AABBCollisions); err != nil {
			log.Println("error while drawing AABBs: ", err, "removing entity")
			var invalidComponentsErr *ecscommon.ErrorMissingComponentDependency
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawCollisions(screen, g.camera, g.tickState.Collisions, g.world.Transforms); err != nil {
			log.Println("error while drawing collisions: ", err)
		}

		proximateEntitiesCount := 0
		for _, others := range g.tickState.ProximateEntities {
			for range others {
				proximateEntitiesCount++
			}
		}

		vel := g.world.Velocities[ecscommon.EntityId(0)].Vector.Length()
		if vel > max_vel {
			max_vel = vel
		}

		ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %f\nTick: %d\nVel: %v\nMaxVel: %f\nMaxPosDiff: %v\nSHG Cells: %v\nProximate Pairs: %d\nResolved Collisions: %d", ebiten.ActualFPS(), g.tickIdx, g.world.Velocities[ecscommon.EntityId(0)].Vector, max_vel, max_pos_diff, g.tickState.CollisionGrid, proximateEntitiesCount, resolvedCollisions))
	}
}

func main() {
	ebiten.SetVsyncEnabled(false)

	g.inputLog = make(map[uint64]map[ecscommon.PlayerId]ecscommon.InputState)
	g.tickState = *ecscommon.NewTickState()

	// f, err := os.ReadFile("replay.json")
	// if err != nil {
	// 	log.Println("error reading replay file: ", err)
	// } else {
	// 	err = json.Unmarshal(f, &replay)
	// 	if err != nil {
	// 		log.Println("error unmarshalling replay file: ", err)
	// 	}
	// }

	pE := g.world.AddEntity()
	g.world.PlayerEntities["player 1"] = pE

	g.world.InputConfigs["player 1"] = &ecscommon.InputConfig{
		Up:              ebiten.KeyW,
		Down:            ebiten.KeyS,
		Left:            ebiten.KeyA,
		Right:           ebiten.KeyD,
		InputSourceFunc: LocalInputSource,
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

	g.world.Parents[pE] = pParComp
	g.world.Children[pE] = pChiComp
	g.world.Transforms[pE] = pTraComp
	g.world.Velocities[pE] = pVelComp
	g.world.Sprites[pE] = pSprComp
	g.world.Colliders[pE] = pColComp

	e := g.world.AddEntity()

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
