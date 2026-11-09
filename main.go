package main

import (
	"ebittest/assetmanager"
	"ebittest/content"
	"ebittest/content/enemyspawning"
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
	"ebittest/ecs/systems/visionsystem"
	"ebittest/tilesystem"
	"ebittest/utils"
	"ebittest/world"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"runtime/pprof"
	"slices"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	g = game{ecs: ecs.NewECSContainer(), zoom: data.DefaultZoom, timescalePercent: 100}

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
	lightOverlay  drawsystem.LightOverlay
	inventoryMenu drawsystem.InventoryMenu
	placer        drawsystem.BlueprintPlacer
	areaPainter   drawsystem.AreaPainter
	// From a click the menu, the placer or the area painter took until the mouse is let go, so that the press doesn't
	// carry on into the world
	uiHasMouse bool
	// Kept from the latest frame for the next tick, since a frame doesn't always run one
	pendingInventory ecs.InventoryAction
	pendingPlacement ecs.BlueprintPlacement
	pendingRemoval   ecs.BlueprintRemoval
	pendingArea      ecs.AreaMarking
	pendingStorage   ecs.StorageMarking
	pendingPickUp    ecs.ItemPickUp
	uiTakesHand      bool
	menuOverSlot     bool

	showFPS       bool
	frameTimes    frameTimes
	showColliders bool
	showChunks    bool
	showBody      bool
	showPaths     bool
	touching      touchingColliders
	// Only drawn without light and fog, so that the simulation, what's remembered included, plays out the same
	fullBright bool
	// The player's vision update last remembered, since remembering the same again changes nothing
	rememberedVision  *ecs.Vision
	rememberedUpdates uint64

	// Simulated time per real time, which isn't saved
	timescalePercent int
	// Each frame adds the timescale, and each tick run takes 100, so that 10% runs a tick every tenth frame and 200%
	// two every frame
	tickBudgetPercent int
	timescaleText     debugText

	// In screen pixels, which are device pixels
	screenSize utils.Vec2f
	// Window pixels per world pixel, from data.MinZoom to data.MaxZoom
	zoom int
	// Screen pixels per UI pixel
	uiScale float64
	// The world is drawn at its own resolution and scaled onto the screen, so that it can be zoomed without changing how
	// anything in it is drawn
	world    *ebiten.Image
	fpsText  debugText
	bodyText debugText
}

// The size of a character in ebitenutil's debug font
const debugGlyphWidth, debugGlyphHeight = 6, 16

// The time from one frame to the next, which includes the updates between them, so that a hitch in either shows
type frameTimes struct {
	last        time.Time
	windowStart time.Time
	windowMax   time.Duration
	// The slowest frame of the last full window, so that the number holds still long enough to read
	shownMax time.Duration
}

func (x *frameTimes) frame(now time.Time) {
	if !x.last.IsZero() {
		x.windowMax = max(x.windowMax, now.Sub(x.last))
	}
	x.last = now

	if now.Sub(x.windowStart) >= data.DebugFrameTimeWindow {
		x.shownMax, x.windowMax, x.windowStart = x.windowMax, 0, now
	}
}

func (x *frameTimes) slowestMs() float64 {
	return float64(x.shownMax) / float64(time.Millisecond)
}

// Which colliders of each kind touched something in the last tick, for drawing them
type touchingColliders struct {
	physics, hitboxes, hurtboxes map[common.EntityId]bool
	features                     map[utils.Vec2i]bool
}

func (g *game) Update() error {
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
		pInputConfig[ecs.Input_All] = ecs.NewKeyboardInputKey(ebiten.KeyShift, -1)
		for i, key := range hotbarKeys {
			pInputConfig[ecs.Input_SelectSlot1+ecs.InputType(i)] = ecs.NewKeyboardInputKey(key, -1)
		}
		pInpComp := ecs.NewInputComponent(pInputConfig, inputsources.InputType_Human, nil, ecs.Facing_Mouse)
		g.ecs.AddComponent(g.playerEntity, pInpComp)
		gamepadFound = true
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
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) || g.inventoryMenu.IsOpen() && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.inventoryMenu.Toggle()
		if g.inventoryMenu.IsOpen() {
			g.placer.Stop()
			g.areaPainter.Stop()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.areaPainter.Toggle()
		if g.areaPainter.IsActive() {
			g.placer.Stop()
			if g.inventoryMenu.IsOpen() {
				g.inventoryMenu.Toggle()
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		g.placer.Toggle(g.ecs.BlueprintAtlas())
		g.startedBuilding()
	}
	g.updateBlueprintCopy()
	g.updateAreaPainter()
	g.updateBlueprintPlacer()
	g.updateInventoryMenu()
	g.updateBlueprintRemoval()
	g.updatePickUp()
	if in, ok := g.ecs.Inputs.Get(g.playerEntity); ok {
		_, wheelUp := ebiten.Wheel()
		in.AddWheel(-wheelUp)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) {
		g.timescalePercent = max(g.timescalePercent-data.TimescaleStepPercent, 0)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		g.timescalePercent = min(g.timescalePercent+data.TimescaleStepPercent, data.MaxTimescalePercent)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.showFPS = !g.showFPS
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.showColliders = !g.showColliders
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.showChunks = !g.showChunks
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		g.showBody = !g.showBody
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.showPaths = !g.showPaths
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		g.fullBright = !g.fullBright
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		g.ecs.SetUntargetable(g.playerEntity, !g.ecs.IsUntargetable(g.playerEntity))
	}

	g.tickBudgetPercent += g.timescalePercent
	for g.tickBudgetPercent >= 100 {
		g.tickBudgetPercent -= 100
		g.tick()
	}

	if g.ecs.CameraFollow {
		pWorldPos, err := g.ecs.WorldPos(g.playerEntity)
		if err != nil {
			log.Println("error getting player ecs position for camera follow: ", err)
		}

		g.ecs.Camera = pWorldPos.Subtract(g.screenSize.Multiply(0.5 / g.ecs.Zoom))
	}

	return nil
}

// One step of the simulation, data.TickMs long
func (g *game) tick() {
	var err error

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
			for _, colony := range g.ecs.Colonies.GetEntities() {
				for range data.ColonistsPerColony {
					colonist, err := templates.SpawnColonist(g.ecs, colony)
					if err != nil {
						log.Fatal(err)
					}
					if err := g.moveToNewWorldSpawn(colonist); err != nil {
						log.Fatal("error moving the colonist to where it can stand: ", err)
					}
				}
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
	if in, ok := tickInputs[g.playerEntity]; ok {
		tickInputs[g.playerEntity] = g.withInventoryMenu(in)
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
	// Where things have moved to, before they run into the doors
	g.chunkContainer.TickDoors(g.ecs)

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
	if vision, ok := g.ecs.Visions.Get(g.playerEntity); ok && (vision != g.rememberedVision || vision.Updates() != g.rememberedUpdates) {
		g.chunkContainer.RememberSeenTiles(vision)
		g.rememberedVision, g.rememberedUpdates = vision, vision.Updates()
	}

	// After vision, so that nothing spawns where the player sees this tick
	if err := enemyspawning.Tick(g.ecs, g.playerEntity); err != nil {
		log.Println("error spawning enemies: ", err)
	}

	if err := g.ecs.RemoveScheduledEntities(); err != nil {
		log.Println("error while removing scheduled entities: ", err)
	}
	// After what's removed lays down what it held, which can keep the harvest area there
	g.ecs.ClearExhaustedHarvest()

	if g.showColliders {
		g.touching = touchingCollidersOf(physicsCollisions, tileCollisions, damageCollisions, terrainDamageCollisions)
	}

	if g.ecs.TickIdx%(data.WorldSavePeriodSec*data.TPS) == 0 {
		g.startWorldSave()
	}

	g.ecs.TickIdx++
}

var hotbarKeys = []ebiten.Key{
	ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5,
	ebiten.KeyDigit6, ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9,
}

// In UI pixels
func (g *game) uiCursor() utils.Vec2f {
	x, y := ebiten.CursorPosition()
	return utils.Vec2f{X: float64(x), Y: float64(y)}.Multiply(1 / g.uiScale)
}

// Turns clicks and hotbar keys in the inventory menu into an action for the next tick. Until that tick has run, which
// at a slow or stopped timescale can take many frames, the menu takes no new actions, since it decides what a click
// does from the inventory as it is.
func (g *game) updateInventoryMenu() {
	inv, ok := g.ecs.Inventories.Get(g.playerEntity)
	if !ok {
		return
	}
	cursor, screen := g.uiCursor(), g.screenSize.Multiply(1/g.uiScale)
	canAct := g.pendingInventory.Kind == ecs.InventoryAction_None

	covers := g.inventoryMenu.Covers(cursor, screen, inv, g.assetmanager)
	if covers && (inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)) {
		g.uiHasMouse = true
		if canAct && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.pendingInventory = g.inventoryMenu.Click(cursor, screen, inv, g.assetmanager)
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		g.uiHasMouse = false
	}

	g.menuOverSlot = g.inventoryMenu.SlotAt(cursor, screen, inv, g.assetmanager) != -1
	if g.menuOverSlot {
		for i, key := range hotbarKeys {
			if inpututil.IsKeyJustPressed(key) && g.pendingInventory.Kind == ecs.InventoryAction_None {
				g.pendingInventory = g.inventoryMenu.HotbarKey(i, cursor, screen, inv, g.assetmanager)
			}
		}
	}

	g.uiTakesHand = g.uiHasMouse || covers || g.placer.IsActive() || g.areaPainter.IsActive()
}

// A click on the panel chooses the area, and a drag elsewhere puts the rectangle from where it started to where it
// ended in the area, or takes it out with shift held as it ends, as the next tick's input. M, escape or a right click
// closes the menu.
func (g *game) updateAreaPainter() {
	if !g.areaPainter.IsActive() {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.uiHasMouse = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
		g.areaPainter.Stop()
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.uiHasMouse = true
		cursor, screen := g.uiCursor(), g.screenSize.Multiply(1/g.uiScale)
		if area, ok := g.areaPainter.AreaAt(cursor, screen, g.assetmanager); ok {
			g.areaPainter.Choose(area)
		} else if !g.areaPainter.Covers(cursor, screen, g.assetmanager) {
			g.areaPainter.Press(g.cursorTile())
		}
	}
	if g.areaPainter.IsDragging() && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		marking := g.areaPainter.Release(g.cursorTile(), ebiten.IsKeyPressed(ebiten.KeyShift))
		if g.pendingArea.Area == data.Area_None {
			g.pendingArea = marking
		}
	}
}

// Build mode, the placer, is on with nothing else: no inventory menu and no area menu
func (g *game) startedBuilding() {
	if !g.placer.IsActive() {
		return
	}
	g.areaPainter.Stop()
	if g.inventoryMenu.IsOpen() {
		g.inventoryMenu.Toggle()
	}
}

// A middle click on a seen tile that a blueprint builds, or on a site, starts build mode with that blueprint. What's
// remembered of the tile is what's copied, so that copying can't find out what's changed out of sight.
func (g *game) updateBlueprintCopy() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		return
	}
	cursor, screen := g.uiCursor(), g.screenSize.Multiply(1/g.uiScale)
	if inv, ok := g.ecs.Inventories.Get(g.playerEntity); ok && g.inventoryMenu.Covers(cursor, screen, inv, g.assetmanager) ||
		g.areaPainter.Covers(cursor, screen, g.assetmanager) {
		return
	}
	tile := g.cursorTile()
	remembered, seen := g.chunkContainer.RecallTile(tile)
	if !seen {
		return
	}
	kind, ok := g.ecs.BlueprintAtlas().ForTile(remembered.Top())
	if site, onSite := g.chunkContainer.Site(tile); onSite {
		kind, ok = site.Blueprint, true
	}
	if !ok {
		return
	}
	g.placer.Start(kind)
	g.startedBuilding()
}

// A click on the panel chooses what to build. Elsewhere, blueprints are put down where they fit as a drag ends, on the
// tile clicked or along the outline of the rectangle dragged out, as the next tick's input, which leaves build mode
// unless shift is held. Storage is marked out by dragging, like an area, and taken away with shift held as the drag
// ends, staying in build mode. B, escape or a right click leaves it.
func (g *game) updateBlueprintPlacer() {
	if !g.placer.IsActive() {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.uiHasMouse = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
		g.placer.Stop()
		return
	}
	if g.placer.IsDragging() && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.releaseBuildDrag()
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	g.uiHasMouse = true
	cursor, screen := g.uiCursor(), g.screenSize.Multiply(1/g.uiScale)
	if g.placer.Covers(cursor, screen, g.ecs.BlueprintAtlas(), g.assetmanager) {
		if chosen, ok := g.placer.BlueprintAt(cursor, screen, g.ecs.BlueprintAtlas(), g.assetmanager); ok {
			g.placer.Start(chosen)
		} else if g.placer.StorageAt(cursor, screen, g.ecs.BlueprintAtlas(), g.assetmanager) {
			g.placer.StartStorage()
		}
		return
	}
	g.placer.Press(g.cursorTile())
}

func (g *game) releaseBuildDrag() {
	tile, shift := g.cursorTile(), ebiten.IsKeyPressed(ebiten.KeyShift)
	if g.placer.IsStorage() {
		if marking := g.placer.Release(tile, shift, g.canPlaceStorage); !g.pendingStorage.Mark {
			g.pendingStorage = marking
		}
		return
	}
	placement := g.placer.ReleaseBlueprints(tile, g.canPlaceBlueprint)
	if g.pendingPlacement.Kind != data.Blueprint_None || placement.Kind == data.Blueprint_None {
		return
	}
	g.pendingPlacement = placement
	if !shift {
		g.placer.Stop()
	}
}

// For the colonist under the cursor while debug info is shown, otherwise what lies on the tile under the cursor, or the
// site there, unless the cursor is on a menu or building. While debug info is shown, it starts with the tile's
// coordinates, and shows them on their own over anything else.
func (g *game) drawSiteTooltip(screen *ebiten.Image) error {
	inv, ok := g.ecs.Inventories.Get(g.playerEntity)
	if !ok || g.placer.IsActive() {
		return nil
	}
	cursor := g.uiCursor()
	if g.inventoryMenu.Covers(cursor, g.screenSize.Multiply(1/g.uiScale), inv, g.assetmanager) ||
		g.areaPainter.Covers(cursor, g.screenSize.Multiply(1/g.uiScale), g.assetmanager) {
		return nil
	}
	tile := g.cursorTile()
	lines := g.tileTooltip(tile)
	if g.showFPS {
		lines = append(append([]string{fmt.Sprintf("Tile %d,%d", tile.X, tile.Y)}, g.chunkContainer.RegionDebugLines(tile)...), lines...)
	}
	if len(lines) == 0 {
		return nil
	}
	return drawsystem.DrawTooltip(screen, g.uiScale, cursor, lines, g.assetmanager)
}

func (g *game) tileTooltip(tile utils.Vec2i) []string {
	if g.showFPS {
		if lines, ok := g.colonistUnderCursor(); ok {
			return lines
		}
	}
	_, seen := g.chunkContainer.RecallTile(tile)
	if !seen {
		return nil
	}
	if stack, ok := g.chunkContainer.GroundStack(tile); ok {
		return drawsystem.GroundStackTooltip(stack, g.ecs)
	}
	site, ok := g.chunkContainer.Site(tile)
	if !ok {
		return nil
	}
	def, ok := g.ecs.BlueprintAtlas().Def(site.Blueprint)
	if !ok {
		return nil
	}
	lines := drawsystem.SiteTooltip(site, def, g.ecs.ItemAtlas(), g.chunkContainer.InArea(data.Area_Home, tile))
	if g.showFPS && site.Order.Id != 0 {
		lines = append(lines, fmt.Sprintf("Order %d, #%d", site.Order.Id, site.Order.Seq))
	}
	return lines
}

// Only on what's been seen, in sight now or remembered, so that placing can't find out what's in unexplored ground
func (g *game) canPlaceBlueprint(tile utils.Vec2i) bool {
	_, seen := g.chunkContainer.RecallTile(tile)
	kind, _ := g.placer.Kind()
	return seen && g.chunkContainer.CanPlaceBlueprint(tile, kind)
}

// Only on what's been seen, like a blueprint
func (g *game) canPlaceStorage(tile utils.Vec2i) bool {
	_, seen := g.chunkContainer.RecallTile(tile)
	return seen && g.ecs.CanPlaceStorage(tile)
}

// R over a blueprint that has nothing delivered yet takes it away, on what's been seen like placing
func (g *game) updateBlueprintRemoval() {
	if !inpututil.IsKeyJustPressed(ebiten.KeyR) || g.pendingRemoval.Remove {
		return
	}
	cursor, screen := g.uiCursor(), g.screenSize.Multiply(1/g.uiScale)
	if inv, ok := g.ecs.Inventories.Get(g.playerEntity); ok && g.inventoryMenu.Covers(cursor, screen, inv, g.assetmanager) ||
		g.placer.Covers(cursor, screen, g.ecs.BlueprintAtlas(), g.assetmanager) || g.areaPainter.Covers(cursor, screen, g.assetmanager) {
		return
	}
	tile := g.cursorTile()
	site, ok := g.chunkContainer.Site(tile)
	if _, seen := g.chunkContainer.RecallTile(tile); !ok || !seen || !site.IsBlueprint() {
		return
	}
	g.pendingRemoval = ecs.BlueprintRemoval{Remove: true, Tile: tile}
}

// Holding the right button down picks up what's on each tile the cursor passes over, as the next tick's input, unless
// the press was a menu's, the cursor is on the inventory menu, or building or marking out areas, where right clicks
// leave
func (g *game) updatePickUp() {
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) || g.uiHasMouse || g.placer.IsActive() || g.areaPainter.IsActive() {
		return
	}
	cursor, screen := g.uiCursor(), g.screenSize.Multiply(1/g.uiScale)
	if inv, ok := g.ecs.Inventories.Get(g.playerEntity); ok && g.inventoryMenu.Covers(cursor, screen, inv, g.assetmanager) {
		return
	}
	if tile := g.cursorTile(); !g.pendingPickUp.PickUp {
		if _, onGround := g.chunkContainer.GroundStack(tile); onGround {
			g.pendingPickUp = ecs.ItemPickUp{PickUp: true, Tile: tile}
		}
	}
}

func (g *game) cursorTile() utils.Vec2i {
	return utils.WorldPosToTilePos(g.cursorWorldPos())
}

func (g *game) cursorWorldPos() utils.Vec2f {
	x, y := ebiten.CursorPosition()
	return utils.Vec2f{X: float64(x), Y: float64(y)}.Multiply(1 / g.ecs.Zoom).Add(g.ecs.Camera)
}

// The debug lines of the colonist closest to the cursor, within half a tile of it
func (g *game) colonistUnderCursor() ([]string, bool) {
	cursor := g.cursorWorldPos()
	best, bestDist := common.EntityId(-1), float64(data.TileSize/2)
	for e, in := range ecs.Query1(&g.ecs.Inputs, ecs.Without(&g.ecs.Disabled)) {
		pos, err := g.ecs.WorldPos(e)
		if err != nil || in.Type() != inputsources.InputType_Colonist {
			continue
		}
		if pos.Subtract(cursor).Length() <= bestDist {
			best, bestDist = e, pos.Subtract(cursor).Length()
		}
	}
	if best == -1 {
		return nil, false
	}
	return inputsources.ColonistDebugLines(best, g.ecs)
}

// Adds the inventory menu's and the placer's pending actions, keeps clicks on the menu or while placing from also using
// what's in hand, and hotbar keys over the menu's slots from also selecting
func (g *game) withInventoryMenu(in ecs.InputState) ecs.InputState {
	in.Inventory, g.pendingInventory = g.pendingInventory, ecs.InventoryAction{}
	in.PlaceBlueprint, g.pendingPlacement = g.pendingPlacement, ecs.BlueprintPlacement{}
	in.RemoveBlueprint, g.pendingRemoval = g.pendingRemoval, ecs.BlueprintRemoval{}
	in.Area, g.pendingArea = g.pendingArea, ecs.AreaMarking{}
	in.Storage, g.pendingStorage = g.pendingStorage, ecs.StorageMarking{}
	in.PickUp, g.pendingPickUp = g.pendingPickUp, ecs.ItemPickUp{}
	if g.menuOverSlot {
		in.SelectSlot = 0
	}
	if g.uiTakesHand {
		in.MainHandEqAbility1, in.MainHandEqAbility2 = 0, 0
		in.OffHandEqAbility1, in.OffHandEqAbility2 = 0, 0
	}
	return in
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

// The scaled world covers the whole screen, which is why the screen isn't cleared every frame
func (g *game) Draw(screen *ebiten.Image) {
	g.frameTimes.frame(time.Now())

	if g.loading {
		screen.Fill(color.RGBA{0, 0, 0, 255})
		g.fpsText.draw(screen, "Loading...", g.uiScale, corner_TopLeft)
		return
	}

	world := g.worldImage(screen)
	g.drawWorld(world)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(g.ecs.Zoom, g.ecs.Zoom)
	screen.DrawImage(world, op)

	isVisible := func(utils.Vec2f) bool { return true }
	if vision, ok := g.ecs.Visions.Get(g.playerEntity); ok && !g.fullBright {
		isVisible = func(pos utils.Vec2f) bool { return drawsystem.Revealed(pos, vision) }
	}
	if err := drawsystem.DrawFloatingTexts(screen, g.uiScale, isVisible, g.ecs); err != nil {
		log.Println("error while drawing floating texts: ", err)
	}

	if err := drawsystem.DrawHotbar(screen, g.uiScale, g.playerEntity, g.ecs, g.assetmanager); err != nil {
		log.Println("error while drawing hotbar: ", err)
	}
	if err := g.inventoryMenu.Draw(screen, g.uiScale, g.uiCursor(), g.playerEntity, g.ecs, g.assetmanager); err != nil {
		log.Println("error while drawing the inventory: ", err)
	}
	if err := g.areaPainter.DrawPanel(screen, g.uiScale, g.uiCursor(), g.assetmanager); err != nil {
		log.Println("error while drawing the area menu: ", err)
	}
	if err := g.placer.DrawPanel(screen, g.uiScale, g.uiCursor(), g.ecs.BlueprintAtlas(), g.ecs.ItemAtlas(), g.assetmanager); err != nil {
		log.Println("error while drawing the blueprint panel: ", err)
	}
	if err := g.drawSiteTooltip(screen); err != nil {
		log.Println("error while drawing the tooltip of a construction site: ", err)
	}

	if g.showFPS {
		g.fpsText.draw(screen, fmt.Sprintf("FPS: %f\nSlowest frame: %.2f ms", ebiten.ActualFPS(), g.frameTimes.slowestMs()), g.uiScale, corner_TopLeft)
	}
	if g.showBody {
		if status, err := drawsystem.BodyStatusText(g.playerEntity, g.ecs); err == nil {
			g.bodyText.draw(screen, status, g.uiScale, corner_BottomRight)
		}
	}
	if g.timescalePercent != 100 {
		g.timescaleText.draw(screen, fmt.Sprintf("Speed: %d%%", g.timescalePercent), g.uiScale, corner_TopRight)
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
		// Unmanaged, so that it isn't put on Ebitengine's shared atlas, whose pages grow to fit it and don't shrink back
		g.world = ebiten.NewImageWithOptions(image.Rect(0, 0, w, h), &ebiten.NewImageOptions{Unmanaged: true})
	}
	return g.world
}

func (g *game) drawWorld(world *ebiten.Image) {
	world.Fill(color.RGBA{0, 0, 0, 255})

	tilesystem.DrawChunks(world, g.ecs.Camera, g.chunkContainer)

	// On the ground, under what's on it and the light and fog, except while building or marking out areas, when it's
	// drawn above the fog with them
	if !g.placer.IsActive() && !g.areaPainter.IsActive() {
		drawsystem.DrawArea(world, g.ecs.Camera, data.Area_Storage, g.chunkContainer.InArea)
	}
	window := drawsystem.DrawWindow(g.ecs.Camera, drawsystem.ImageSize(world))
	staticSprites := slices.Concat(
		g.chunkContainer.FeatureSprites(window, g.ecs.TickIdx*data.TickMs),
		drawsystem.GroundItemSprites(window, g.ecs),
	)
	if err := drawsystem.DrawSprites(world, g.ecs.Camera, staticSprites, g.ecs, g.assetmanager); err != nil {
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
	inEmergency := func(e common.EntityId) bool { return inputsources.InEmergency(e, g.ecs) }
	if err := drawsystem.DrawAlerts(world, g.ecs.Camera, inEmergency, g.ecs, g.assetmanager); err != nil {
		log.Println("error while drawing emergencies: ", err)
	}

	if vision, ok := g.ecs.Visions.Get(g.playerEntity); ok && !g.fullBright {
		g.lightOverlay.Draw(world, g.ecs.Camera, vision.Light())
		g.visionOverlay.Draw(world, g.ecs.Camera, vision, g.drawMemory)
	}
	// Above everything, even what stands on them, so that what's planned looks the same in the light, the dark and the fog
	blueprints := g.chunkContainer.BlueprintSprites(drawsystem.DrawWindow(g.ecs.Camera, drawsystem.ImageSize(world)))
	if err := drawsystem.DrawStaticSprites(world, g.ecs.Camera, blueprints, g.assetmanager); err != nil {
		log.Println("error while drawing blueprints: ", err)
	}

	// Above the fog, since blueprints and areas can go anywhere. A click while placing or marking out an area doesn't
	// use what's in hand, so its target isn't shown, and nor is the blueprint under the panel, which a click would
	// choose from rather than put down.
	_, placing := g.placer.Kind()
	if g.areaPainter.IsActive() {
		for _, area := range drawsystem.Areas() {
			drawsystem.DrawArea(world, g.ecs.Camera, area, g.chunkContainer.InArea)
		}
		drawsystem.DrawArea(world, g.ecs.Camera, data.Area_Storage, g.chunkContainer.InArea)
	} else if g.placer.IsActive() {
		drawsystem.DrawArea(world, g.ecs.Camera, data.Area_Home, g.chunkContainer.InArea)
		drawsystem.DrawArea(world, g.ecs.Camera, data.Area_Storage, g.chunkContainer.InArea)
	}
	if g.areaPainter.IsActive() {
		g.areaPainter.Draw(world, g.ecs.Camera, g.cursorTile(), ebiten.IsKeyPressed(ebiten.KeyShift))
	} else if g.placer.IsStorage() && g.placer.IsDragging() {
		g.placer.DrawStorage(world, g.ecs.Camera, g.cursorTile(), ebiten.IsKeyPressed(ebiten.KeyShift), g.canPlaceStorage)
	} else if g.placer.IsActive() {
		tile := g.cursorTile()
		overPanel := g.placer.Covers(g.uiCursor(), g.screenSize.Multiply(1/g.uiScale), g.ecs.BlueprintAtlas(), g.assetmanager)
		if placing && (!overPanel || g.placer.IsDragging()) {
			if err := g.placer.Draw(world, g.ecs.Camera, tile, g.canPlaceBlueprint, g.ecs.BlueprintAtlas(), g.assetmanager); err != nil {
				log.Println("error while drawing the blueprint being placed: ", err)
			}
		}
	} else if err := drawsystem.DrawTileTarget(world, g.ecs.Camera, g.playerEntity, g.ecs); err != nil {
		log.Println("error while drawing the tile target: ", err)
	}

	if g.showColliders {
		g.drawColliders(world)
	}
	if g.showChunks {
		g.chunkContainer.DrawChunkBorders(world, g.ecs.Camera)
	}
	if g.showPaths {
		drawsystem.DrawPaths(world, g.ecs.Camera, g.ecs)
	}
}

// Grown to fit the longest text yet, so that it isn't reallocated as the text changes
type debugText struct {
	image *ebiten.Image
}

// The debug font is tiny at the screen's resolution, so the text is printed at UI pixels and scaled up. Only the text's
// own area is drawn, since a full-screen draw for a few characters costs as much as drawing the world.
type screenCorner uint8

const (
	corner_TopLeft screenCorner = iota
	corner_TopRight
	corner_BottomRight
)

func (d *debugText) draw(screen *ebiten.Image, msg string, uiScale float64, corner screenCorner) {
	lines := strings.Split(msg, "\n")
	longest := 0
	for _, l := range lines {
		longest = max(longest, len(l))
	}
	// ebitenutil draws a shadow one pixel down and to the right of the text
	w, h := longest*debugGlyphWidth+1, len(lines)*debugGlyphHeight+1

	if d.image == nil || d.image.Bounds().Dx() < w || d.image.Bounds().Dy() < h {
		allocW, allocH := w, h
		if d.image != nil {
			allocW, allocH = max(w, d.image.Bounds().Dx()), max(h, d.image.Bounds().Dy())
			d.image.Deallocate()
		}
		d.image = ebiten.NewImage(allocW, allocH)
	}

	d.image.Clear()
	ebitenutil.DebugPrint(d.image, msg)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(uiScale, uiScale)
	size := screen.Bounds().Size()
	switch corner {
	case corner_TopRight:
		op.GeoM.Translate(float64(size.X)-float64(w)*uiScale, 0)
	case corner_BottomRight:
		op.GeoM.Translate(float64(size.X)-float64(w)*uiScale, float64(size.Y)-float64(h)*uiScale)
	}
	screen.DrawImage(d.image.SubImage(image.Rect(0, 0, w, h)).(*ebiten.Image), op)
}

// Entities are never remembered, only the terrain and the features on it
func (g *game) drawMemory(dst *ebiten.Image, camera utils.Vec2f) {
	g.chunkContainer.DrawMemory(dst, camera, func(dst *ebiten.Image, camera utils.Vec2f, sprites []common.StaticSprite) {
		if err := drawsystem.DrawStaticSprites(dst, camera, sprites, g.assetmanager); err != nil {
			log.Println("error while drawing remembered features: ", err)
		}
	})
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
	cpuProfile := flag.String("cpuprofile", "", "write a CPU profile of the game's run to this file")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(data.TPS)

	assets, err := assetmanager.LoadAssetManagerFromJson("assets/assets.json")
	if err != nil {
		log.Fatal("error loading assets: ", err)
	}
	if err := assets.CheckTags(common.CodeAssetTags); err != nil {
		log.Fatal("error checking the assets code uses: ", err)
	}
	g.assetmanager = assets
	g.ecs.SetAssets(assets)

	// Content first, since the item atlas names its abilities
	if err := content.Register(g.ecs); err != nil {
		log.Fatal("error registering content: ", err)
	}

	bodyAtlas, err := ecs.LoadBodyAtlasFromJson("assets/bodies/bodies.json")
	if err != nil {
		log.Fatal("error loading body atlas from json: ", err)
	}
	g.ecs.SetBodyAtlas(bodyAtlas)

	itemAtlas, err := ecs.LoadItemAtlasFromJson("assets/items/items.json")
	if err != nil {
		log.Fatal("error loading item atlas from json: ", err)
	}
	if err := g.ecs.SetItemAtlas(itemAtlas); err != nil {
		log.Fatal("error setting item atlas: ", err)
	}
	if err := itemAtlas.CheckImageAssets(assets); err != nil {
		log.Fatal("error checking the item atlas against the assets: ", err)
	}

	g.chunkContainer = tilesystem.NewChunkContainer()
	if err := g.chunkContainer.GenerateTileAtlasFromJson("assets/tiles/tiles.json", itemAtlas, assets); err != nil {
		log.Fatal("error generating tile atlas from json: ", err)
	}
	blueprintAtlas, err := ecs.LoadBlueprintAtlasFromJson("assets/blueprints/blueprints.json", itemAtlas, assets)
	if err != nil {
		log.Fatal("error loading blueprint atlas from json: ", err)
	}
	if err := g.chunkContainer.GenerateFeatureAtlasFromJson("assets/features/features.json", itemAtlas, assets); err != nil {
		log.Fatal("error generating feature atlas from json: ", err)
	}
	if err := g.chunkContainer.SetBlueprintAtlas(blueprintAtlas); err != nil {
		log.Fatal("error checking the blueprint atlas against the tile and feature atlases: ", err)
	}
	g.ecs.SetBlueprintAtlas(blueprintAtlas)
	if err := itemAtlas.CheckPlacedFeatures(g.chunkContainer.HasFeature); err != nil {
		log.Fatal("error checking the item atlas against the feature atlas: ", err)
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
		if _, err := templates.SpawnColonyCenter(g.ecs); err != nil {
			log.Fatal(err)
		}
		g.newWorld = true
	}

	g.loading = true

	ebiten.SetWindowSize(1920, 1080)
	ebiten.SetWindowTitle("ebittest")
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetScreenClearedEveryFrame(false)

	stopProfile := startCPUProfile(*cpuProfile)
	err = ebiten.RunGame(&g)
	// Stopped before a fatal error exits, which would otherwise leave the profile unwritten
	stopProfile()
	if err != nil {
		log.Fatal(err)
	}
}

// Does nothing without a path
func startCPUProfile(path string) (stop func()) {
	if path == "" {
		return func() {}
	}

	f, err := os.Create(path)
	if err != nil {
		log.Fatal("error creating CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("error starting CPU profile: ", err)
	}

	return func() {
		pprof.StopCPUProfile()
		if err := f.Close(); err != nil {
			log.Println("error closing CPU profile: ", err)
		}
	}
}
