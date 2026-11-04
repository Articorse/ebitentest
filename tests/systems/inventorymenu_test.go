package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
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
		Y: (menuScreen.Y-gridH)/2 + float64(row*(menuSlotPx+data.HotbarSlotGapPx)) + menuSlotPx/2,
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
	covers := func(at utils.Vec2f) bool { return menu.Covers(at, menuScreen, inv, testutil.Assets(t)) }
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
