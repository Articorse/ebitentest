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
	"ebittest/ecs/systems/itemsystem"
	"ebittest/ecs/systems/movementsystem"
	"ebittest/ecs/systems/platformsystem"
	"ebittest/ecs/systems/timersystem"
	"ebittest/ecs/systems/visionsystem"
	"ebittest/tilesystem"
	"ebittest/utils"
	"ebittest/world"
	"errors"
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	g = game{ecs: ecs.NewECSContainer(), assetmanager: assetmanager.NewAssetManager(), zoom: data.DefaultZoom}

	physicsColliders  = ecs.Colliders(&g.ecs.PhysicsColliders)
	hurtboxColliders  = ecs.Colliders(&g.ecs.HurtboxColliders)
	hitboxColliders   = ecs.Colliders(&g.ecs.HitboxColliders)
	platformColliders = ecs.Colliders(&g.ecs.PlatformColliders)

	gamepadFound = false
)

type game struct {
	ecs          *ecs.ECSContainer
	playerEntity common.EntityId

	chunkContainer *tilesystem.ChunkContainer
	loading        bool
	justLoaded     bool
	newWorld       bool

	assetmanager  *assetmanager.AssetManager
	visionOverlay drawsystem.VisionOverlay

	showFPS       bool
	showColliders bool
	touching      touchingColliders

	// In screen pixels, which are device pixels
	screenSize utils.Vec2f
	// Window pixels per world pixel, from data.MinZoom to data.MaxZoom
	zoom int
	// Screen pixels per UI pixel
	uiScale float64
	// The world is drawn at its own resolution and scaled onto the screen, so that it can be zoomed without changing how
	// anything in it is drawn
	world *ebiten.Image
	// The debug font is tiny at the screen's resolution, so debug text is printed here and scaled up
	debugText *ebiten.Image
}

// Which colliders of each kind touched something in the last tick, for drawing them
type touchingColliders struct {
	physics, hitboxes, hurtboxes map[common.EntityId]bool
	features                     map[utils.Vec2i]bool
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
		pInputConfig[ecs.Input_DropItem] = ecs.NewKeyboardInputKey(ebiten.KeyQ, -1)
		for i, key := range []ebiten.Key{
			ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5,
			ebiten.KeyDigit6, ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9,
		} {
			pInputConfig[ecs.Input_SelectSlot1+ecs.InputType(i)] = ecs.NewKeyboardInputKey(key, -1)
		}
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
	for eid, in := range ecs.Query1(&g.ecs.Inputs, ecs.Without(&g.ecs.Disabled)) {
		inputParams := in.Params()

		inputSourceFunc, err := g.ecs.InputSourceFunc(in.Type())
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

	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		g.setZoom(g.zoom + 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		g.setZoom(g.zoom - 1)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.ecs.CameraFollow = !g.ecs.CameraFollow
	}
	if g.ecs.CameraFollow {
		pWorldPos, err := g.ecs.WorldPos(g.playerEntity)
		if err != nil {
			log.Println("error getting player ecs position for camera follow: ", err)
		}

		g.ecs.Camera = pWorldPos.Subtract(g.screenSize.Multiply(0.5 / g.ecs.Zoom))
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.showFPS = !g.showFPS
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.showColliders = !g.showColliders
	}

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

	physicsAABBCollisions := collisionsystem.GetAABBCollisions(physicsColliders, proximateEntities, g.ecs)
	physicsCollisions := collisionsystem.GetCollisions(physicsColliders, physicsColliders, physicsAABBCollisions, g.ecs)

	_, err = collisionsystem.ResolvePhysicsCollisions(physicsCollisions, g.ecs)
	if err != nil {
		log.Println("error during collision resolution: ", err)
	}

	potentialCollisionTiles, err := g.chunkContainer.GetTilesWithPotentialPhysicsCollisions(g.ecs)
	if err != nil {
		log.Println("error during getting tiles with potential collisions: ", err)
	}

	tileAABBCollisions, err := tilesystem.GetAABBCollisions(physicsColliders, potentialCollisionTiles, g.ecs)
	if err != nil {
		log.Println("error during tile AABB collision checking: ", err)
	}

	tileCollisions, err := tilesystem.GetCollisions(physicsColliders, tileAABBCollisions, g.ecs)
	if err != nil {
		log.Println("error during tile collision checking: ", err)
	}

	_, err = tilesystem.ResolveTileCollisions(tileCollisions, g.ecs)
	if err != nil {
		log.Println("error during tile collision resolution: ", err)
	}

	itemsystem.Tick(g.ecs)

	err = abilitysystem.Tick(g.ecs)
	if err != nil {
		log.Println("ability system tick error: ", err)
	}

	damageAABBCollisions := collisionsystem.GetDirectedAABBCollisions(hurtboxColliders, hitboxColliders, proximateEntities, g.ecs)
	damageCollisions := collisionsystem.GetCollisions(hurtboxColliders, hitboxColliders, damageAABBCollisions, g.ecs)

	damagesystem.Tick(g.ecs)

	terrainDamageCollisions, err := g.chunkContainer.GetHurtboxTileCollisions(g.ecs)
	if err != nil {
		log.Println("error during terrain damage collision checking: ", err)
	}

	_, err = damagesystem.DealContactDamage(damageCollisions, terrainDamageCollisions, g.ecs)
	if err != nil {
		log.Println("error during dealing contact damage: ", err)
	}

	err = platformsystem.Tick(g.ecs.TickState.CollisionGrid, g.ecs)
	if err != nil {
		log.Println("platform system tick error: ", err)
	}

	visionsystem.Tick(g.ecs)
	if vision, ok := g.ecs.Visions.Get(g.playerEntity); ok {
		if pos, err := g.ecs.WorldPos(g.playerEntity); err == nil {
			g.chunkContainer.RememberSeenTiles(pos, vision)
		}
	}

	if err := g.ecs.RemoveScheduledEntities(); err != nil {
		log.Println("error while removing scheduled entities: ", err)
	}

	if g.showColliders {
		g.touching = touchingCollidersOf(physicsCollisions, tileCollisions, damageCollisions, terrainDamageCollisions)
	}

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

	header, err := world.Snapshot(g.chunkContainer.Seed(), g.playerEntity, g.ecs)
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

	header, err := world.Snapshot(g.chunkContainer.Seed(), g.playerEntity, g.ecs)
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
		g.drawDebugText(screen, "Loading...")
		return
	}

	world := g.worldImage(screen)
	g.drawWorld(world)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(g.ecs.Zoom, g.ecs.Zoom)
	screen.DrawImage(world, op)

	isVisible := func(utils.Vec2f) bool { return true }
	if vision, ok := g.ecs.Visions.Get(g.playerEntity); ok {
		if center, err := g.ecs.WorldPos(g.playerEntity); err == nil {
			isVisible = func(pos utils.Vec2f) bool { return drawsystem.Revealed(pos, center, vision) }
		}
	}
	if err := drawsystem.DrawFloatingTexts(screen, g.uiScale, isVisible, g.ecs); err != nil {
		log.Println("error while drawing floating texts: ", err)
	}

	if err := drawsystem.DrawHotbar(screen, g.uiScale, g.playerEntity, g.ecs, g.assetmanager); err != nil {
		log.Println("error while drawing hotbar: ", err)
	}

	if g.showFPS {
		g.drawDebugText(screen, fmt.Sprintf("FPS: %f", ebiten.ActualFPS()))
	}
}

// Big enough to cover the screen once scaled up by the zoom
func (g *game) worldImage(screen *ebiten.Image) *ebiten.Image {
	size := screen.Bounds().Size()
	w := int(math.Ceil(float64(size.X) / g.ecs.Zoom))
	h := int(math.Ceil(float64(size.Y) / g.ecs.Zoom))
	if g.world == nil || g.world.Bounds().Dx() != w || g.world.Bounds().Dy() != h {
		if g.world != nil {
			g.world.Deallocate()
		}
		g.world = ebiten.NewImage(w, h)
	}
	return g.world
}

func (g *game) drawWorld(world *ebiten.Image) {
	world.Fill(color.RGBA{0, 0, 0, 255})

	if err := tilesystem.DrawChunks(world, g.ecs.Camera, g.chunkContainer); err != nil {
		log.Println("error while drawing chunks: ", err)
	}

	if err := drawsystem.DrawSprites(
		world,
		g.ecs.Camera,
		g.chunkContainer.FeatureSprites(drawsystem.DrawWindow(g.ecs.Camera, drawsystem.ImageSize(world))),
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

	if vision, ok := g.ecs.Visions.Get(g.playerEntity); ok {
		if pos, err := g.ecs.WorldPos(g.playerEntity); err == nil {
			g.visionOverlay.Draw(world, g.ecs.Camera, pos, vision, g.drawMemory)
		}
	}

	if err := drawsystem.DrawTileTarget(world, g.ecs.Camera, g.playerEntity, g.ecs); err != nil {
		log.Println("error while drawing the tile target: ", err)
	}

	if g.showColliders {
		g.drawColliders(world)
	}
}

func (g *game) drawDebugText(screen *ebiten.Image, msg string) {
	size := screen.Bounds().Size()
	w := int(math.Ceil(float64(size.X) / g.uiScale))
	h := int(math.Ceil(float64(size.Y) / g.uiScale))
	if g.debugText == nil || g.debugText.Bounds().Dx() != w || g.debugText.Bounds().Dy() != h {
		if g.debugText != nil {
			g.debugText.Deallocate()
		}
		g.debugText = ebiten.NewImage(w, h)
	}

	g.debugText.Clear()
	ebitenutil.DebugPrint(g.debugText, msg)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(g.uiScale, g.uiScale)
	screen.DrawImage(g.debugText, op)
}

// Entities are never remembered, only the terrain and the features on it
func (g *game) drawMemory(dst *ebiten.Image, camera utils.Vec2f) {
	g.chunkContainer.DrawMemory(dst, camera)

	features := g.chunkContainer.RememberedFeatureSprites(drawsystem.DrawWindow(camera, drawsystem.ImageSize(dst)))
	if err := drawsystem.DrawStaticSprites(dst, camera, features, data.MemoryBrightness, g.assetmanager); err != nil {
		log.Println("error while drawing remembered features: ", err)
	}
}

// Keeps what's at the center of the screen there, so that zooming with the camera not following the player doesn't
// pull the view towards the top left
func (g *game) setZoom(zoom int) {
	zoom = min(max(zoom, data.MinZoom), data.MaxZoom)
	if zoom == g.zoom {
		return
	}

	newZoom := g.ecs.Zoom * float64(zoom) / float64(g.zoom)
	center := g.ecs.Camera.Add(g.screenSize.Multiply(0.5 / g.ecs.Zoom))
	g.ecs.Camera = center.Subtract(g.screenSize.Multiply(0.5 / newZoom))
	g.ecs.Zoom = newZoom
	g.zoom = zoom
}

// At the screen's real resolution, so that the world and the UI are each scaled up on their own
func (g *game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	deviceScale := ebiten.Monitor().DeviceScaleFactor()
	g.ecs.Zoom = float64(g.zoom) * deviceScale
	g.uiScale = data.UIScale * deviceScale
	g.screenSize = utils.Vec2f{
		X: math.Ceil(float64(outsideWidth) * deviceScale),
		Y: math.Ceil(float64(outsideHeight) * deviceScale),
	}
	return int(g.screenSize.X), int(g.screenSize.Y)
}

func (g *game) drawColliders(world *ebiten.Image) {
	kinds := []struct {
		colliders      ecs.ColliderSet
		color, touches color.RGBA
		touching       map[common.EntityId]bool
	}{
		// Platforms resolve what lands on them themselves, without reporting it, so they're never shown touching
		{platformColliders, data.Debug_PlatformColliderColor, data.Debug_PlatformColliderColor, nil},
		{physicsColliders, data.Debug_PhysicsColliderColor, data.Debug_PhysicsColliderTouchingColor, g.touching.physics},
		{hitboxColliders, data.Debug_HitboxColliderColor, data.Debug_HitboxColliderTouchingColor, g.touching.hitboxes},
		{hurtboxColliders, data.Debug_HurtboxColliderColor, data.Debug_HurtboxColliderTouchingColor, g.touching.hurtboxes},
	}

	// Touching outlines go over the rest, so that one isn't hidden under another collider in the same place, such as
	// the player's hitbox over its physics collider
	for _, touching := range []bool{false, true} {
		featureColor := data.Debug_FeatureColliderColor
		if touching {
			featureColor = data.Debug_FeatureColliderTouchingColor
		}
		g.chunkContainer.DrawFeatureColliders(world, g.ecs.Camera, featureColor, func(t utils.Vec2i) bool {
			return g.touching.features[t] == touching
		})

		for _, k := range kinds {
			lineColor := k.color
			if touching {
				lineColor = k.touches
			}
			collisionsystem.DrawColliders(k.colliders, lineColor, func(e common.EntityId) bool {
				return k.touching[e] == touching
			}, world, g.ecs.Camera, g.ecs)
		}
	}
}

func touchingCollidersOf(
	physicsCollisions []common.Collision,
	physicsTileCollisions []common.TileCollision,
	damageCollisions []common.Collision,
	damageTileCollisions []common.TileCollision,
) touchingColliders {
	t := touchingColliders{
		physics:   map[common.EntityId]bool{},
		hitboxes:  map[common.EntityId]bool{},
		hurtboxes: map[common.EntityId]bool{},
		features:  map[utils.Vec2i]bool{},
	}
	for _, c := range physicsCollisions {
		t.physics[c.A], t.physics[c.B] = true, true
	}
	for _, c := range physicsTileCollisions {
		t.physics[c.E] = true
		t.features[c.Tile] = t.features[c.Tile] || c.Feature
	}
	for _, c := range damageCollisions {
		t.hurtboxes[c.A], t.hitboxes[c.B] = true, true
	}
	for _, c := range damageTileCollisions {
		t.hurtboxes[c.E] = true
		t.features[c.Tile] = t.features[c.Tile] || c.Feature
	}
	return t
}

func (g *game) moveToNewWorldSpawn(e common.EntityId) error {
	pos, err := g.ecs.WorldPos(e)
	if err != nil {
		return err
	}

	spawn, ok := g.chunkContainer.NewWorldSpawnPos(pos)
	if !ok {
		log.Printf("no passable tile within %d tiles of %v, spawning entity %d there anyway", data.WorldgenSpawnSearchRadiusTiles, pos, e)
		return nil
	}

	return g.ecs.SetWorldPos(e, spawn)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(data.TPS)

	// Content first, since the item atlas names its abilities
	if err := content.Register(g.ecs); err != nil {
		log.Fatal("error registering content: ", err)
	}

	itemAtlas, err := ecs.LoadItemAtlasFromJson("assets/items/items.json")
	if err != nil {
		log.Fatal("error loading item atlas from json: ", err)
	}
	if err := g.ecs.SetItemAtlas(itemAtlas); err != nil {
		log.Fatal("error setting item atlas: ", err)
	}

	g.chunkContainer = tilesystem.NewChunkContainer()
	if err := g.chunkContainer.GenerateTileAtlasFromJson("assets/tiles/atlas.json", itemAtlas); err != nil {
		log.Fatal("error generating tile atlas from json: ", err)
	}
	if err := itemAtlas.CheckPlacedTiles(g.chunkContainer.HasTile); err != nil {
		log.Fatal("error checking the item atlas against the tile atlas: ", err)
	}
	if err := g.chunkContainer.GenerateFeatureAtlasFromJson("assets/features/features.json", itemAtlas); err != nil {
		log.Fatal("error generating feature atlas from json: ", err)
	}

	g.ecs.TickState = *common.NewTickState()
	g.ecs.SetSimulatedPosCheck(g.chunkContainer.IsActiveWorldPos)
	g.ecs.SetTerrain(g.chunkContainer)

	header, worldExists, err := world.Load(g.ecs)
	if err != nil {
		log.Fatal("error loading world: ", err)
	}

	if worldExists {
		if err := tilesystem.RecoverChunkFiles(header.Generation); err != nil {
			log.Fatal("error recovering chunk files: ", err)
		}
		g.chunkContainer.SetGeneration(header.Generation)
		g.chunkContainer.SetSeed(header.Seed)
		if err := g.chunkContainer.LoadMemory(); err != nil {
			log.Fatal("error loading memory: ", err)
		}
		g.playerEntity = header.PlayerEntity
	} else {
		if err := world.Reset(); err != nil {
			log.Fatal("error resetting world: ", err)
		}
		seed := world.NewSeed()
		g.chunkContainer.SetSeed(seed)
		g.ecs.SeedRng(seed)
		g.playerEntity, err = templates.SpawnPlayer(g.ecs)
		if err != nil {
			log.Fatal("error spawning player: ", err)
		}
		if err := g.moveToNewWorldSpawn(g.playerEntity); err != nil {
			log.Fatal("error moving player to spawn: ", err)
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
