package main

import (
	"ebittest/ecs"
	"ebittest/ecs/components"
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

	DEBUG = true
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
		g.camera = pTraComp.Pos.Subtract(utils.Vec2{X: float64(width) / 2, Y: float64(height) / 2})
	}

	// DEBUG: For testing purposes only
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		maps.DeleteFunc(g.world.Transforms,
			func(k ecscommon.EntityId, _ *components.Transform) bool { return k == pE })
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		DEBUG = !DEBUG
	}

	g.tickState = *ecscommon.NewTickState()

	if err := movementsystem.Tick(g.world.Velocities, g.world.Transforms); err != nil {
		log.Println("movement system error: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponent
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	g.tickState.CollisionGrid, err = commonsystems.PopulateSpatialHashGrid(g.world.Transforms)
	if err != nil {
		log.Println("error during populating spatial hash grid: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponent
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	proximateEntities, err := collisionsystem.GetSHGProximities(g.tickState.CollisionGrid, g.world.Colliders, g.world.Transforms)
	if err != nil {
		log.Println("error during spatial hash grid proximity checking: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponent
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	aabbcollisions, err := collisionsystem.GetAABBCollisions(proximateEntities, g.world.Colliders, g.world.Transforms)
	if err != nil {
		log.Println("error during AABB collision checking: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponent
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	collisions, err := collisionsystem.GetCollisions(aabbcollisions, g.world.Colliders, g.world.Transforms)
	if err != nil {
		log.Println("error during collision checking: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponent
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
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
		log.Println("error while drawing frame: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponent
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	// DEBUG
	if DEBUG {
		if err := collisionsystem.DrawColliders(screen, g.camera, g.world.Colliders, g.world.Transforms, g.tickState.Collisions); err != nil {
			log.Println("error while drawing colliders: ", err, "removing entity")
			var invalidComponentsErr *ecscommon.ErrorMissingComponent
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawAABBs(screen, g.camera, g.world.Colliders, g.world.Transforms, g.tickState.AABBCollisions); err != nil {
			log.Println("error while drawing AABBs: ", err, "removing entity")
			var invalidComponentsErr *ecscommon.ErrorMissingComponent
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		proximateEntitiesCount := 0
		for _, others := range g.tickState.ProximateEntities {
			for range others {
				proximateEntitiesCount++
			}
		}

		ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %f\nTick: %d", ebiten.ActualFPS(), g.tickIdx))
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SHG Cells: %v\nProximate Pairs: %d", g.tickState.CollisionGrid, proximateEntitiesCount), 0, 50)
	}
	// END DEBUG
}

func (g *game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return width, height
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
	pTraComp := components.NewTransformComponent()
	pVelComp := components.NewVelocityComponent()
	pSprComp := components.NewSpriteComponent()

	pSpr, _, err := ebitenutil.NewImageFromFile("assets/sprites/slime.png")
	if err != nil {
		log.Fatal(err)
	}

	pSprComp.Image = pSpr

	pColComp, err := components.NewColliderComponent(
		[]utils.Vec2{
			utils.Vec2{X: -20, Y: -20},
			utils.Vec2{X: 25, Y: -25},
			utils.Vec2{X: 20, Y: 20},
			utils.Vec2{X: -25, Y: 25},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	g.world.Parents[pE] = pParComp
	g.world.Children[pE] = pChiComp
	g.world.Transforms[pE] = pTraComp
	g.world.Velocities[pE] = pVelComp
	g.world.Sprites[pE] = pSprComp
	g.world.Colliders[pE] = pColComp

	e := g.world.AddEntity()

	eParComp := components.NewParentComponent()
	eChiComp := components.NewChildrenComponent()
	eTraComp := components.NewTransformComponent()
	eTraComp.Pos.X = 450
	eTraComp.Pos.Y = 250
	eVelComp := components.NewVelocityComponent()
	eSprComp := components.NewSpriteComponent()

	eSpr, _, err := ebitenutil.NewImageFromFile("assets/sprites/tree.png")
	if err != nil {
		log.Fatal(err)
	}

	eSprComp.Image = eSpr

	eColComp, err := components.NewColliderComponent(
		[]utils.Vec2{
			utils.Vec2{X: -20, Y: -20},
			utils.Vec2{X: 25, Y: -25},
			utils.Vec2{X: 20, Y: 20},
			utils.Vec2{X: -25, Y: 25},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

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
