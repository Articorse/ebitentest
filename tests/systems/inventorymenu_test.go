package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

// In UI pixels, with room for the hotbar to the left of the inventory
var menuScreen = utils.Vec2f{X: 640, Y: 360}

const menuSlotPx = 32

func inventorySlotCenter(i int) utils.Vec2f {
	gridW := float64(data.InventoryColumns*menuSlotPx + (data.InventoryColumns-1)*data.HotbarSlotGapPx)
	gridH := float64(data.InventoryRows*menuSlotPx + (data.InventoryRows-1)*data.HotbarSlotGapPx)
	col, row := i%data.InventoryColumns, i/data.InventoryColumns
	return utils.Vec2f{
		X: (menuScreen.X-gridW)/2 + float64(col*(menuSlotPx+data.HotbarSlotGapPx)) + menuSlotPx/2,
		Y: menuScreen.Y - data.InventoryMarginPx - data.InventoryPanelPaddingPx - gridH + float64(row*(menuSlotPx+data.HotbarSlotGapPx)) + menuSlotPx/2,
	}
}

func hotbarSlotCenter(h int) utils.Vec2f {
	total := float64(data.HotbarSlots*menuSlotPx + (data.HotbarSlots-1)*data.HotbarSlotGapPx)
	return utils.Vec2f{
		X: data.HotbarMarginPx + menuSlotPx/2,
		Y: (menuScreen.Y-total)/2 + float64(h*(menuSlotPx+data.HotbarSlotGapPx)) + menuSlotPx/2,
	}
}

func newMenu(t *testing.T) (*drawsystem.InventoryMenu, *ecs.ECSContainer, *ecs.Inventory, func(utils.Vec2f) ecs.InventoryAction) {
	t.Helper()
	x, player, _, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	assets := testutil.Assets(t)
	menu := &drawsystem.InventoryMenu{}
	menu.Toggle()
	return menu, x, inv, func(at utils.Vec2f) ecs.InventoryAction { return menu.Click(at, menuScreen, inv, assets) }
}

func TestClickingAnItemAndThenASlotMovesIt(t *testing.T) {
	_, _, _, click := newMenu(t)

	if a := click(inventorySlotCenter(0)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("picking up the mattock did %v, want nothing until it's put down", a)
	}
	if a := click(inventorySlotCenter(20)); a != (ecs.InventoryAction{Kind: ecs.InventoryAction_Move, From: 0, To: 20}) {
		t.Fatalf("putting the mattock down did %v, want it moved from slot 0 to 20", a)
	}
	if a := click(inventorySlotCenter(21)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("clicking an empty slot after putting the mattock down did %v", a)
	}
	if a := click(inventorySlotCenter(22)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("clicking an empty slot picked up nothing, which was then moved: %v", a)
	}
}

func TestPuttingAnItemBackOrClosingTheMenuLeavesItWhereItWas(t *testing.T) {
	menu, _, _, click := newMenu(t)

	click(inventorySlotCenter(1))
	if a := click(inventorySlotCenter(1)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("putting the bazooka back did %v", a)
	}
	if a := click(inventorySlotCenter(20)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("the bazooka was still held after being put back: %v", a)
	}

	click(inventorySlotCenter(1))
	menu.Toggle()
	menu.Toggle()
	if a := click(inventorySlotCenter(20)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("the bazooka was still held after closing the menu: %v", a)
	}
}

func TestClickingAHotbarSlotBindsWhatsHeldOrUnbindsIt(t *testing.T) {
	_, x, inv, click := newMenu(t)

	click(inventorySlotCenter(2))
	if a := click(hotbarSlotCenter(7)); a != (ecs.InventoryAction{Kind: ecs.InventoryAction_Bind, From: 2, To: 7}) {
		t.Fatalf("putting the gun on the hotbar did %v, want slot 2 bound to hotbar slot 7", a)
	}
	if a := click(hotbarSlotCenter(0)); a != (ecs.InventoryAction{Kind: ecs.InventoryAction_Unbind, To: 0}) {
		t.Fatalf("clicking the mattock's hotbar slot with nothing held did %v, want it unbound", a)
	}

	if err := x.BindHotbar(x.Root(slotItem(inv, 0)), 0, -1); err != nil {
		t.Fatal(err)
	}
	if a := click(hotbarSlotCenter(0)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("clicking an empty hotbar slot did %v", a)
	}
}

func TestAHeldItemThatLeftItsSlotIsLetGo(t *testing.T) {
	_, x, inv, click := newMenu(t)
	player := x.Root(slotItem(inv, 0))

	click(inventorySlotCenter(0))
	if err := x.MoveInventoryItem(player, 0, 30); err != nil {
		t.Fatal(err)
	}
	if a := click(inventorySlotCenter(20)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("the slot the mattock left was still held: %v", a)
	}
}

func TestAClosedMenuIgnoresClicks(t *testing.T) {
	menu, _, inv, click := newMenu(t)
	menu.Toggle()

	if a := click(inventorySlotCenter(0)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("a click on the closed menu did %v", a)
	}
	if menu.Covers(inventorySlotCenter(0), menuScreen, inv, testutil.Assets(t)) {
		t.Fatal("the closed menu takes clicks")
	}
}

func TestTheMenuTakesClicksOverItAndAnywhereWhileHolding(t *testing.T) {
	menu, _, inv, click := newMenu(t)
	covers := func(at utils.Vec2f) bool {
		return menu.Covers(at, menuScreen, inv, testutil.Assets(t))
	}
	corner := inventorySlotCenter(0).Subtract(utils.Vec2f{X: menuSlotPx/2 + data.InventoryPanelPaddingPx - 1, Y: menuSlotPx/2 + data.InventoryPanelPaddingPx - 1})
	away := utils.Vec2f{X: menuScreen.X - 1, Y: 1}

	if !covers(inventorySlotCenter(35)) || !covers(corner) || !covers(hotbarSlotCenter(8)) {
		t.Fatal("the menu doesn't take clicks over its slots, its border or the hotbar")
	}
	right := inventorySlotCenter(data.InventoryColumns - 1).Add(utils.Vec2f{X: menuSlotPx/2 + data.InventoryPanelPaddingPx + 1})
	below := inventorySlotCenter(data.InventoryColumns*data.InventoryRows - 1).Add(utils.Vec2f{Y: menuSlotPx/2 + data.InventoryPanelPaddingPx + 1})
	if covers(corner.Subtract(utils.Vec2f{X: 2})) || covers(right) || covers(below) || covers(away) {
		t.Fatal("the menu takes clicks away from it")
	}

	click(inventorySlotCenter(30))
	if covers(away) {
		t.Fatal("clicking an empty slot left the menu taking clicks away from it")
	}

	click(inventorySlotCenter(0))
	if !covers(away) {
		t.Fatal("the menu doesn't take clicks away from it while an item is held")
	}
}

func TestPuttingAnItemDownOnAnotherPicksThatOneUp(t *testing.T) {
	menu, x, inv, click := newMenu(t)
	player := x.Root(slotItem(inv, 0))
	bazooka := slotItem(inv, 1)

	click(inventorySlotCenter(0))
	a := click(inventorySlotCenter(1))
	if a != (ecs.InventoryAction{Kind: ecs.InventoryAction_Move, From: 0, To: 1}) {
		t.Fatalf("putting the mattock on the bazooka did %v, want them swapped", a)
	}
	act(t, x, player, a.Kind, a.From, a.To)

	if slotItem(inv, 0) != bazooka || !menu.Covers(utils.Vec2f{X: menuScreen.X - 1, Y: 1}, menuScreen, inv, testutil.Assets(t)) {
		t.Fatal("the bazooka wasn't picked up in the mattock's place")
	}
	if a := click(inventorySlotCenter(20)); a != (ecs.InventoryAction{Kind: ecs.InventoryAction_Move, From: 0, To: 20}) {
		t.Fatalf("putting the bazooka down did %v, want it moved from the slot the mattock came from", a)
	}
}

func TestAStackPutOnAnotherKeepsWhatDidntFitHeld(t *testing.T) {
	menu, x, inv, click := newMenu(t)
	player := x.Root(slotItem(inv, 0))
	away := utils.Vec2f{X: menuScreen.X - 1, Y: 1}
	covers := func() bool { return menu.Covers(away, menuScreen, inv, testutil.Assets(t)) }
	putStack(t, x, player, 10, 50)
	putStack(t, x, player, 11, 70)
	putStack(t, x, player, 12, 10)

	click(inventorySlotCenter(10))
	a := click(inventorySlotCenter(11))
	act(t, x, player, a.Kind, a.From, a.To)
	if countOf(x, slotItem(inv, 10)) != 20 || !covers() {
		t.Fatal("the 20 wood that didn't fit aren't still held")
	}

	a = click(inventorySlotCenter(12))
	act(t, x, player, a.Kind, a.From, a.To)
	if slotItem(inv, 10) != -1 || covers() {
		t.Fatal("the menu still holds the wood that all went onto the stack")
	}
	if a := click(inventorySlotCenter(20)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("clicking an empty slot after the held wood were used up did %v", a)
	}
}

func TestHotbarKeysOverASlotBindOrUnbind(t *testing.T) {
	menu, _, inv, _ := newMenu(t)
	assets := testutil.Assets(t)

	if a := menu.HotbarKey(6, inventorySlotCenter(2), menuScreen, inv, assets); a != (ecs.InventoryAction{Kind: ecs.InventoryAction_Bind, From: 2, To: 6}) {
		t.Fatalf("7 over the gun did %v, want it bound to hotbar slot 7", a)
	}
	if a := menu.HotbarKey(0, inventorySlotCenter(30), menuScreen, inv, assets); a != (ecs.InventoryAction{Kind: ecs.InventoryAction_Unbind, To: 0}) {
		t.Fatalf("1 over an empty slot did %v, want hotbar slot 1 unbound", a)
	}
	if a := menu.HotbarKey(0, hotbarSlotCenter(3), menuScreen, inv, assets); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("1 away from the inventory's slots did %v", a)
	}
	if menu.SlotAt(inventorySlotCenter(2), menuScreen, inv, assets) != 2 {
		t.Fatal("the slot under the cursor isn't found")
	}

	menu.Toggle()
	if a := menu.HotbarKey(6, inventorySlotCenter(2), menuScreen, inv, assets); a.Kind != ecs.InventoryAction_None || menu.SlotAt(inventorySlotCenter(2), menuScreen, inv, assets) != -1 {
		t.Fatalf("a hotbar key over the closed menu did %v", a)
	}
}

func TestTheInventorySitsAtTheBottomOfTheScreen(t *testing.T) {
	menu, _, inv, _ := newMenu(t)
	bottom := inventorySlotCenter(data.InventoryColumns * data.InventoryRows / 2)
	bottom.Y = menuScreen.Y - data.InventoryMarginPx - 1
	if !menu.Covers(bottom, menuScreen, inv, testutil.Assets(t)) || menu.Covers(utils.Vec2f{X: bottom.X, Y: menuScreen.Y - data.InventoryMarginPx}, menuScreen, inv, testutil.Assets(t)) {
		t.Fatal("the panel doesn't end its margin above the bottom of the screen")
	}
}

// With the game's one blueprint, its panel is a single slot in the bottom right corner
func blueprintSlotCenter() utils.Vec2f {
	inset := float64(data.InventoryMarginPx + data.InventoryPanelPaddingPx + menuSlotPx/2)
	return utils.Vec2f{X: menuScreen.X - inset, Y: menuScreen.Y - inset}
}

func TestBlueprintsAreChosenFromTheirPanelInTheBottomRightCornerWhileBuilding(t *testing.T) {
	var placer drawsystem.BlueprintPlacer
	blueprints, assets := testutil.Blueprints(t), testutil.Assets(t)
	if placer.Covers(blueprintSlotCenter(), menuScreen, blueprints, assets) {
		t.Fatal("the blueprint panel takes clicks outside build mode")
	}
	if _, ok := placer.BlueprintAt(blueprintSlotCenter(), menuScreen, blueprints, assets); ok {
		t.Fatal("a blueprint can be chosen outside build mode")
	}

	placer.Start(data.Blueprint_WoodWall)
	wall := blueprintSlotCenter().Subtract(utils.Vec2f{X: menuSlotPx + data.HotbarSlotGapPx})
	if kind, ok := placer.BlueprintAt(wall, menuScreen, blueprints, assets); !ok || kind != data.Blueprint_WoodWall {
		t.Fatalf("the blueprint panel has %v (%v) in its first slot, want wooden walls", kind, ok)
	}
	if !placer.StorageAt(blueprintSlotCenter(), menuScreen, blueprints, assets) || placer.StorageAt(wall, menuScreen, blueprints, assets) {
		t.Fatal("storage isn't the panel's last slot, after the blueprints")
	}
	if _, ok := placer.BlueprintAt(blueprintSlotCenter(), menuScreen, blueprints, assets); ok {
		t.Fatal("the storage slot counts as a blueprint")
	}
	edge := utils.Vec2f{X: menuScreen.X - data.InventoryMarginPx - 1, Y: menuScreen.Y - data.InventoryMarginPx - 1}
	if !placer.Covers(edge, menuScreen, blueprints, assets) || placer.Covers(edge.Add(utils.Vec2f{X: 2}), menuScreen, blueprints, assets) ||
		placer.Covers(edge.Add(utils.Vec2f{Y: 2}), menuScreen, blueprints, assets) {
		t.Fatal("the panel doesn't take clicks on its border, or takes them in the margin past it")
	}
	if _, ok := placer.BlueprintAt(edge, menuScreen, blueprints, assets); ok {
		t.Fatal("the panel's border counts as its blueprint")
	}
}

func TestTheInventoryMenuHasNoBlueprints(t *testing.T) {
	menu, _, inv, click := newMenu(t)
	if menu.Covers(blueprintSlotCenter(), menuScreen, inv, testutil.Assets(t)) {
		t.Fatal("the inventory menu takes clicks where the blueprint panel goes")
	}
	if a := click(blueprintSlotCenter()); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("clicking where the blueprint panel goes did %v to the inventory", a)
	}
}

func TestBuildModeStartsWithTheFirstBlueprintAndTogglesOff(t *testing.T) {
	var placer drawsystem.BlueprintPlacer
	blueprints := testutil.Blueprints(t)
	placer.Toggle(blueprints)
	if kind, ok := placer.Kind(); !ok || kind != blueprints.Kinds()[0] {
		t.Fatalf("build mode started with %v (%v), want the first blueprint", kind, ok)
	}
	placer.Toggle(blueprints)
	if _, ok := placer.Kind(); ok {
		t.Fatal("toggling build mode again left it on")
	}
}

func TestATileIsCopiedAsTheBlueprintThatBuildsIt(t *testing.T) {
	blueprints := testutil.Blueprints(t)
	def, _ := blueprints.Def(data.Blueprint_WoodWall)
	if kind, ok := blueprints.ForTile(def.Tile()); !ok || kind != data.Blueprint_WoodWall {
		t.Fatalf("the wooden wall tile copies as %v (%v), want wooden walls", kind, ok)
	}
	if kind, ok := blueprints.ForTile(data.Tile_Dirt); ok {
		t.Fatalf("dirt, which nothing builds, copies as %v", kind)
	}
}

func TestABlueprintsTooltipHasItsNameAndCosts(t *testing.T) {
	def, _ := testutil.Blueprints(t).Def(data.Blueprint_WoodWall)
	if lines := drawsystem.BlueprintTooltip(def, testutil.ItemAtlas(t)); !slices.Equal(lines, []string{"Wooden wall", "2 Wood"}) {
		t.Fatalf("the wooden wall's tooltip says %q, want Wooden wall and 2 Wood", lines)
	}
}

func TestThePlacerHoldsTheChosenBlueprintUntilStopped(t *testing.T) {
	var p drawsystem.BlueprintPlacer
	if _, ok := p.Kind(); ok {
		t.Fatal("a new placer is placing")
	}
	p.Start(data.Blueprint_WoodWall)
	if kind, ok := p.Kind(); !ok || kind != data.Blueprint_WoodWall {
		t.Fatal("the placer isn't placing the chosen blueprint")
	}
	p.Stop()
	if _, ok := p.Kind(); ok {
		t.Fatal("the placer is still placing after stopping")
	}
}

func TestASitesTooltipHasWhatsDeliveredAndHowFarBuildingHasCome(t *testing.T) {
	def, _ := testutil.Blueprints(t).Def(data.Blueprint_WoodWall)
	site := ecs.ConstructionSite{Blueprint: data.Blueprint_WoodWall, Delivered: []ecs.ItemCost{{Kind: data.Item_Wood}}, Health: 1, MaxHealth: 100}
	if lines := drawsystem.SiteTooltip(site, def, testutil.ItemAtlas(t), true); !slices.Equal(lines, []string{"Wooden wall", "Wood 0/2", "Construction 0%"}) {
		t.Fatalf("a new site's tooltip says %q", lines)
	}
	site.Delivered[0].Count, site.Health = 2, 34
	if lines := drawsystem.SiteTooltip(site, def, testutil.ItemAtlas(t), true); !slices.Equal(lines, []string{"Wooden wall", "Wood 2/2", "Construction 33%"}) {
		t.Fatalf("a supplied site a third built has a tooltip saying %q", lines)
	}
}

func TestASiteOffHomeSaysColonistsLeaveIt(t *testing.T) {
	def, _ := testutil.Blueprints(t).Def(data.Blueprint_WoodWall)
	site := ecs.ConstructionSite{Blueprint: data.Blueprint_WoodWall, Delivered: []ecs.ItemCost{{Kind: data.Item_Wood}}, Health: 1, MaxHealth: 100}
	if lines := drawsystem.SiteTooltip(site, def, testutil.ItemAtlas(t), false); lines[len(lines)-1] != "Outside the home area" {
		t.Fatalf("a site off home has a tooltip saying %q", lines)
	}
}

func TestDraggingOutAnAreaMarksTheRectangleBetweenItsEnds(t *testing.T) {
	var p drawsystem.AreaPainter
	p.Press(utils.Vec2i{X: 3, Y: 3})
	if p.IsDragging() {
		t.Fatal("dragging started without the area menu open")
	}
	p.Toggle()
	p.Press(utils.Vec2i{X: 5, Y: -2})
	if m := p.Release(utils.Vec2i{X: 1, Y: 4}, false); m != (ecs.AreaMarking{Area: data.Area_Home, First: utils.Vec2i{X: 1, Y: -2}, Last: utils.Vec2i{X: 5, Y: 4}}) {
		t.Fatalf("dragging from 5, -2 to 1, 4 marked %+v", m)
	}
	if m := p.Release(utils.Vec2i{}, false); m.Area != data.Area_None {
		t.Fatalf("releasing without dragging marked %+v", m)
	}
	p.Choose(data.Area_Harvest)
	p.Press(utils.Vec2i{})
	if m := p.Release(utils.Vec2i{}, true); m != (ecs.AreaMarking{Area: data.Area_Harvest, Erase: true}) {
		t.Fatalf("a shift drag over one tile with harvest chosen did %+v, want it erased from harvest", m)
	}
	p.Press(utils.Vec2i{})
	p.Toggle()
	if p.IsActive() || p.IsDragging() {
		t.Fatal("closing the area menu left it open or dragging")
	}
}

func TestTheAreaMenuOpensOnHomeWithItsButtonsInTheBottomRightCorner(t *testing.T) {
	var p drawsystem.AreaPainter
	assets := testutil.Assets(t)
	if p.Covers(blueprintSlotCenter(), menuScreen, assets) {
		t.Fatal("the area menu takes clicks while closed")
	}
	if _, ok := p.AreaAt(blueprintSlotCenter(), menuScreen, assets); ok {
		t.Fatal("an area can be chosen while the menu is closed")
	}

	p.Toggle()
	if area, ok := p.Area(); !ok || area != data.Area_Home {
		t.Fatalf("the area menu opened on %v (%v), want home", area, ok)
	}
	home := blueprintSlotCenter().Subtract(utils.Vec2f{X: menuSlotPx + data.HotbarSlotGapPx})
	if area, ok := p.AreaAt(home, menuScreen, assets); !ok || area != data.Area_Home {
		t.Fatalf("the first button is %v (%v), want home", area, ok)
	}
	if area, ok := p.AreaAt(blueprintSlotCenter(), menuScreen, assets); !ok || area != data.Area_Harvest {
		t.Fatalf("the last button is %v (%v), want harvest", area, ok)
	}
	between := blueprintSlotCenter().Subtract(utils.Vec2f{X: menuSlotPx/2 + data.HotbarSlotGapPx/2.0})
	if _, ok := p.AreaAt(between, menuScreen, assets); ok || !p.Covers(between, menuScreen, assets) {
		t.Fatal("the gap between buttons isn't the panel's without being a button")
	}
}

func TestStorageIsDraggedOutInBuildModeAndStaysChosen(t *testing.T) {
	var placer drawsystem.BlueprintPlacer
	placer.Start(data.Blueprint_WoodWall)
	placer.Press(utils.Vec2i{})
	everywhere := func(utils.Vec2i) bool { return true }
	if m := placer.Release(utils.Vec2i{X: 2}, false, everywhere); m.Mark {
		t.Fatalf("a drag with a blueprint chosen marked out storage: %+v", m)
	}

	placer.StartStorage()
	if _, ok := placer.Kind(); ok || !placer.IsStorage() || !placer.IsActive() {
		t.Fatal("choosing storage didn't leave build mode on with storage and no blueprint chosen")
	}
	v := func(x, y int) utils.Vec2i { return utils.Vec2i{X: x, Y: y} }
	placer.Press(v(3, 1))
	if m := placer.Release(v(2, 2), false, everywhere); !m.Mark || m.Erase || !slices.Equal(m.Tiles, []utils.Vec2i{v(2, 1), v(3, 1), v(2, 2), v(3, 2)}) {
		t.Fatalf("dragging storage from 3, 1 to 2, 2 marked %+v, want its 4 tiles row by row", m)
	}
	if m := placer.Release(utils.Vec2i{}, false, everywhere); m.Mark {
		t.Fatalf("releasing without dragging marked %+v", m)
	}
	notTheCorner := func(t utils.Vec2i) bool { return t != v(0, 0) }
	placer.Press(v(0, 0))
	if m := placer.Release(v(1, 0), false, notTheCorner); !slices.Equal(m.Tiles, []utils.Vec2i{v(1, 0)}) {
		t.Fatalf("a drag over a tile storage can't go on marked %+v, want just the one it can", m)
	}
	placer.Press(v(0, 0))
	if m := placer.Release(v(0, 0), false, func(utils.Vec2i) bool { return false }); m.Mark {
		t.Fatalf("a drag where storage can't go marked %+v", m)
	}
	placer.Press(v(0, 0))
	if m := placer.Release(v(1, 0), true, func(utils.Vec2i) bool { return false }); !m.Mark || !m.Erase || len(m.Tiles) != 2 {
		t.Fatalf("a shift drag did %+v, want storage taken off both tiles, whether it could go there or not", m)
	}
	if !placer.IsStorage() {
		t.Fatal("marking out storage left build mode")
	}
	placer.Press(utils.Vec2i{})
	placer.Toggle(testutil.Blueprints(t))
	if placer.IsActive() || placer.IsDragging() {
		t.Fatal("toggling build mode off left it on or dragging")
	}
}

func TestBlueprintsGoAlongTheOutlineOfWhatsDraggedOut(t *testing.T) {
	var placer drawsystem.BlueprintPlacer
	everywhere := func(utils.Vec2i) bool { return true }
	v := func(x, y int) utils.Vec2i { return utils.Vec2i{X: x, Y: y} }
	placer.Press(v(0, 0))
	if placer.IsDragging() {
		t.Fatal("dragging started outside build mode")
	}
	placer.Start(data.Blueprint_WoodWall)
	if tiles := placer.BlueprintTiles(v(3, 4)); !slices.Equal(tiles, []utils.Vec2i{v(3, 4)}) {
		t.Fatalf("without a drag, a blueprint would go on %v, want just the cursor's tile", tiles)
	}

	for _, c := range []struct {
		what     string
		from, to utils.Vec2i
		want     []utils.Vec2i
	}{
		{"a click", v(2, 2), v(2, 2), []utils.Vec2i{v(2, 2)}},
		{"a drag along a row, backwards", v(4, 1), v(1, 1), []utils.Vec2i{v(1, 1), v(2, 1), v(3, 1), v(4, 1)}},
		{"a drag down a column", v(0, 0), v(0, 2), []utils.Vec2i{v(0, 0), v(0, 1), v(0, 2)}},
		{"a 3×3 drag", v(2, 2), v(0, 0), []utils.Vec2i{v(0, 0), v(1, 0), v(2, 0), v(0, 1), v(2, 1), v(0, 2), v(1, 2), v(2, 2)}},
	} {
		placer.Press(c.from)
		if tiles := placer.BlueprintTiles(c.to); !slices.Equal(tiles, c.want) {
			t.Fatalf("%s would put blueprints on %v, want %v", c.what, tiles, c.want)
		}
		if p := placer.ReleaseBlueprints(c.to, everywhere); p.Kind != data.Blueprint_WoodWall || !slices.Equal(p.Tiles, c.want) {
			t.Fatalf("%s put down %+v, want wooden walls on %v", c.what, p, c.want)
		}
	}

	placer.Press(v(0, 0))
	notTheCorner := func(t utils.Vec2i) bool { return t != v(1, 0) }
	if p := placer.ReleaseBlueprints(v(1, 0), notTheCorner); !slices.Equal(p.Tiles, []utils.Vec2i{v(0, 0)}) {
		t.Fatalf("a drag over a tile a blueprint doesn't fit on put down %+v, want just the one it fits on", p)
	}
	placer.Press(v(0, 0))
	if p := placer.ReleaseBlueprints(v(0, 0), func(utils.Vec2i) bool { return false }); p.Kind != data.Blueprint_None {
		t.Fatalf("a click where nothing fits put down %+v", p)
	}
	if p := placer.ReleaseBlueprints(v(0, 0), everywhere); p.Kind != data.Blueprint_None {
		t.Fatalf("releasing without a drag put down %+v", p)
	}

	placer.StartStorage()
	placer.Press(v(0, 0))
	if p := placer.ReleaseBlueprints(v(2, 0), everywhere); p.Kind != data.Blueprint_None || len(p.Tiles) > 0 {
		t.Fatalf("a storage drag put down blueprints: %+v", p)
	}
}
