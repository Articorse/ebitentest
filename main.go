package main

import (
	"ebittest/ecs"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/ecs/systems/movementsystem"
	"ebittest/utils"
	"errors"
	"fmt"
	"image/color"
	"log"
	"maps"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	g         = game{world: ecs.NewWorld()}
	tickState = ecscommon.NewTickState()

	height = 360
	width  = 640

	DEBUG = true
)

type game struct {
	world   *ecs.World
	tickIdx uint64
	camera  utils.Vec2
}

func (g *game) Update() error {
	// mX, mY := ebiten.CursorPosition()
	// pCenterX := g.p.x + float64(g.p.img.Bounds().Dx())/2
	// pCenterY := g.p.y + float64(g.p.img.Bounds().Dy())/2
	// dX := float64(mX) - pCenterX
	// dY := float64(mY) - pCenterY
	// r = math.Atan2(dY, dX)
	var err error

	if len(g.world.Players) == 0 {
		log.Fatalf("no player entity found")
	}

	pConf, ok := g.world.Players["player 1"]
	if !ok {
		log.Fatalf("'player 1' not found")
	}
	pE := pConf.Entity

	pVelComp, ok := g.world.Velocities[pE]
	if !ok {
		log.Fatalf("player entity does not have a velocity component")
	}

	pTraComp, ok := g.world.Transforms[pE]
	if !ok {
		log.Fatalf("player entity does not have a transform component")
	}

	v := utils.Vec2{X: 0, Y: 0}

	if ebiten.IsKeyPressed(ebiten.KeyA) {
		v.X -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		v.Y -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		v.X += 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		v.Y += 1
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
	if ebiten.IsKeyPressed(ebiten.KeyC) {
		g.camera = pTraComp.Pos.Subtract(utils.Vec2{X: float64(width) / 2, Y: float64(height) / 2})
	}

	// DEBUG: For testing purposes only
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		maps.DeleteFunc(g.world.Transforms,
			func(k ecscommon.Entity, _ *components.Transform) bool { return k == pE })
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		DEBUG = !DEBUG
	}

	v = v.Normalized()

	pVelComp.Vector = pVelComp.Vector.Add(v.Multiply(2))

	tickState = ecscommon.NewTickState()

	if err := movementsystem.Tick(g.world.Velocities, g.world.Transforms); err != nil {
		log.Println("movement system error: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponent
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	tickState.CollisionGrid, err = collisionsystem.PopulateSpatialHashGrid(g.world.Colliders, g.world.Transforms)
	if err != nil {
		log.Println("error during populating spatial hash grid: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponent
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	proximateEntities, err := collisionsystem.GetSHGProximities(tickState.CollisionGrid, g.world.Colliders, g.world.Transforms)
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

	tickState.ProximateEntities = proximateEntities
	tickState.AABBCollisions = aabbcollisions
	tickState.Collisions = collisions

	g.tickIdx++

	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})
	if err := drawsystem.DrawFrame(screen, g.camera, g.world.Sprites, g.world.Transforms); err != nil {
		log.Println("error while drawing frame: ", err, "removing entity")
		var invalidComponentsErr *ecscommon.ErrorMissingComponent
		if errors.As(err, &invalidComponentsErr) {
			g.world.RemoveEntity(invalidComponentsErr.Entity)
		}
	}

	// DEBUG
	if DEBUG {
		if err := collisionsystem.DrawColliders(screen, g.camera, g.world.Colliders, g.world.Transforms, tickState.Collisions); err != nil {
			log.Println("error while drawing colliders: ", err, "removing entity")
			var invalidComponentsErr *ecscommon.ErrorMissingComponent
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		if err := collisionsystem.DrawAABBs(screen, g.camera, g.world.Colliders, g.world.Transforms, tickState.AABBCollisions); err != nil {
			log.Println("error while drawing AABBs: ", err, "removing entity")
			var invalidComponentsErr *ecscommon.ErrorMissingComponent
			if errors.As(err, &invalidComponentsErr) {
				g.world.RemoveEntity(invalidComponentsErr.Entity)
			}
		}

		proximateEntitiesCount := 0
		for _, others := range tickState.ProximateEntities {
			for range others {
				proximateEntitiesCount++
			}
		}

		ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %f\nTick: %d", ebiten.ActualFPS(), g.tickIdx))
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SHG Cells: %v\nProximate Entities: %d", tickState.CollisionGrid, proximateEntitiesCount), 0, 50)
	}
	// END DEBUG
}

func (g *game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return width, height
}

func main() {
	ebiten.SetVsyncEnabled(false)

	pE := g.world.AddEntity()
	pKm := ecscommon.KeyMaps{
		Up:    ebiten.KeyW,
		Down:  ebiten.KeyS,
		Left:  ebiten.KeyA,
		Right: ebiten.KeyD,
	}

	g.world.AddPlayer("player 1", pE, pKm)

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

	eSpr, _, err := ebitenutil.NewImageFromFile("assets/sprites/slime.png")
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
