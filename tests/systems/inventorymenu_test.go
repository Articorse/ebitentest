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
	if menu.Covers(inventorySlotCenter(0), menuScreen, inv, testutil.Blueprints(t), testutil.Assets(t)) {
		t.Fatal("the closed menu takes clicks")
	}
}

func TestTheMenuTakesClicksOverItAndAnywhereWhileHolding(t *testing.T) {
	menu, _, inv, click := newMenu(t)
	covers := func(at utils.Vec2f) bool {
		return menu.Covers(at, menuScreen, inv, testutil.Blueprints(t), testutil.Assets(t))
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

	if slotItem(inv, 0) != bazooka || !menu.Covers(utils.Vec2f{X: menuScreen.X - 1, Y: 1}, menuScreen, inv, testutil.Blueprints(t), testutil.Assets(t)) {
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
	covers := func() bool { return menu.Covers(away, menuScreen, inv, testutil.Blueprints(t), testutil.Assets(t)) }
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
	if !menu.Covers(bottom, menuScreen, inv, testutil.Blueprints(t), testutil.Assets(t)) || menu.Covers(utils.Vec2f{X: bottom.X, Y: menuScreen.Y - data.InventoryMarginPx}, menuScreen, inv, testutil.Blueprints(t), testutil.Assets(t)) {
		t.Fatal("the panel doesn't end its margin above the bottom of the screen")
	}
}

// With the game's one blueprint, its panel is a single slot in the bottom right corner
func blueprintSlotCenter() utils.Vec2f {
	inset := float64(data.InventoryMarginPx + data.InventoryPanelPaddingPx + menuSlotPx/2)
	return utils.Vec2f{X: menuScreen.X - inset, Y: menuScreen.Y - inset}
}

func TestBlueprintsAreChosenFromTheirPanelInTheBottomRightCorner(t *testing.T) {
	menu, _, inv, click := newMenu(t)
	blueprints, assets := testutil.Blueprints(t), testutil.Assets(t)

	if kind, ok := menu.BlueprintAt(blueprintSlotCenter(), menuScreen, blueprints, assets); !ok || kind != data.Blueprint_Planks {
		t.Fatalf("the blueprint panel has %v (%v) in its slot, want planks", kind, ok)
	}
	edge := utils.Vec2f{X: menuScreen.X - data.InventoryMarginPx - 1, Y: menuScreen.Y - data.InventoryMarginPx - 1}
	if !menu.Covers(edge, menuScreen, inv, blueprints, assets) || menu.Covers(edge.Add(utils.Vec2f{X: 2}), menuScreen, inv, blueprints, assets) ||
		menu.Covers(edge.Add(utils.Vec2f{Y: 2}), menuScreen, inv, blueprints, assets) {
		t.Fatal("the menu doesn't take clicks on the blueprint panel's border, or takes them in the margin past it")
	}
	if _, ok := menu.BlueprintAt(edge, menuScreen, blueprints, assets); ok {
		t.Fatal("the panel's border counts as its blueprint")
	}
	if a := click(blueprintSlotCenter()); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("clicking a blueprint did %v to the inventory", a)
	}

	menu.Toggle()
	if _, ok := menu.BlueprintAt(blueprintSlotCenter(), menuScreen, blueprints, assets); ok {
		t.Fatal("a blueprint can be chosen from the closed menu")
	}
}

func TestABlueprintsTooltipHasItsNameAndCosts(t *testing.T) {
	def, _ := testutil.Blueprints(t).Def(data.Blueprint_Planks)
	if lines := drawsystem.BlueprintTooltip(def, testutil.ItemAtlas(t)); !slices.Equal(lines, []string{"Planks", "2 Wood"}) {
		t.Fatalf("the planks' tooltip says %q, want Planks and 2 Wood", lines)
	}
}

func TestThePlacerHoldsTheChosenBlueprintUntilStopped(t *testing.T) {
	var p drawsystem.BlueprintPlacer
	if _, ok := p.Kind(); ok {
		t.Fatal("a new placer is placing")
	}
	p.Start(data.Blueprint_Planks)
	if kind, ok := p.Kind(); !ok || kind != data.Blueprint_Planks {
		t.Fatal("the placer isn't placing the chosen blueprint")
	}
	p.Stop()
	if _, ok := p.Kind(); ok {
		t.Fatal("the placer is still placing after stopping")
	}
}

func TestASitesTooltipHasWhatsDeliveredAndHowFarBuildingHasCome(t *testing.T) {
	def, _ := testutil.Blueprints(t).Def(data.Blueprint_Planks)
	site := ecs.ConstructionSite{Blueprint: data.Blueprint_Planks, Delivered: []ecs.ItemCost{{Kind: data.Item_Wood}}, Health: 1, MaxHealth: 100}
	if lines := drawsystem.SiteTooltip(site, def, testutil.ItemAtlas(t), true); !slices.Equal(lines, []string{"Planks", "Wood 0/2", "Construction 0%"}) {
		t.Fatalf("a new site's tooltip says %q", lines)
	}
	site.Delivered[0].Count, site.Health = 2, 34
	if lines := drawsystem.SiteTooltip(site, def, testutil.ItemAtlas(t), true); !slices.Equal(lines, []string{"Planks", "Wood 2/2", "Construction 33%"}) {
		t.Fatalf("a supplied site a third built has a tooltip saying %q", lines)
	}
}

func TestASiteOffHomeSaysColonistsLeaveIt(t *testing.T) {
	def, _ := testutil.Blueprints(t).Def(data.Blueprint_Planks)
	site := ecs.ConstructionSite{Blueprint: data.Blueprint_Planks, Delivered: []ecs.ItemCost{{Kind: data.Item_Wood}}, Health: 1, MaxHealth: 100}
	if lines := drawsystem.SiteTooltip(site, def, testutil.ItemAtlas(t), false); lines[len(lines)-1] != "Outside the home area" {
		t.Fatalf("a site off home has a tooltip saying %q", lines)
	}
}

func TestDraggingOutHomeMarksTheRectangleBetweenItsEnds(t *testing.T) {
	var p drawsystem.HomePainter
	p.Press(utils.Vec2i{X: 3, Y: 3})
	if p.IsDragging() {
		t.Fatal("dragging started without the painter on")
	}
	p.Toggle()
	p.Press(utils.Vec2i{X: 5, Y: -2})
	if m := p.Release(utils.Vec2i{X: 1, Y: 4}, false); m != (ecs.HomeMarking{Mark: ecs.Home_Mark, First: utils.Vec2i{X: 1, Y: -2}, Last: utils.Vec2i{X: 5, Y: 4}}) {
		t.Fatalf("dragging from 5, -2 to 1, 4 marked %+v", m)
	}
	if m := p.Release(utils.Vec2i{}, false); m.Mark != ecs.Home_None {
		t.Fatalf("releasing without dragging marked %+v", m)
	}
	p.Press(utils.Vec2i{})
	if m := p.Release(utils.Vec2i{}, true); m != (ecs.HomeMarking{Mark: ecs.Home_Unmark}) {
		t.Fatalf("a shift drag over one tile did %+v, want it unmarked", m)
	}
	p.Press(utils.Vec2i{})
	p.Toggle()
	if p.IsActive() || p.IsDragging() {
		t.Fatal("turning the painter off left it on or dragging")
	}
}
