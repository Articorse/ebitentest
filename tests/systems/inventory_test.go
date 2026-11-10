package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func act(t *testing.T, x *ecs.ECSContainer, player common.EntityId, kind ecs.InventoryActionKind, from, to int) {
	t.Helper()
	press(t, x, player, ecs.InputState{Inventory: ecs.InventoryAction{Kind: kind, From: from, To: to}})
	testutil.RemoveScheduled(t, x)
	checkConsistency(t, x)
}

// Of kilograms of wood
func putStack(t *testing.T, x *ecs.ECSContainer, player common.EntityId, slot, kilograms int) common.EntityId {
	t.Helper()
	stack, err := x.SpawnItemStack(data.Item_Wood, kilograms*data.MilligramsPerKg, testutil.WorldPos(t, x, player))
	if err != nil {
		t.Fatal(err)
	}
	if err := x.PutInInventory(player, slot, stack); err != nil {
		t.Fatal(err)
	}
	return stack
}

func hotbarItem(inv *ecs.Inventory, h int) common.EntityId {
	e, ok := inv.HotbarItem(h)
	if !ok {
		return -1
	}
	return e
}

func slotItem(inv *ecs.Inventory, i int) common.EntityId {
	e, ok := inv.Slot(i)
	if !ok {
		return -1
	}
	return e
}

func TestMovedItemsKeepTheirHotbarSlots(t *testing.T) {
	x, player, mattock, bazooka, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)

	act(t, x, player, ecs.InventoryAction_Move, 0, 20)
	if slotItem(inv, 20) != mattock || slotItem(inv, 0) != -1 {
		t.Fatal("the mattock wasn't moved into the empty slot")
	}
	act(t, x, player, ecs.InventoryAction_Move, 20, 1)
	if slotItem(inv, 1) != mattock || slotItem(inv, 20) != bazooka {
		t.Fatal("the mattock and the bazooka weren't swapped")
	}

	if hotbarItem(inv, 0) != mattock || hotbarItem(inv, 1) != bazooka || inHand(x, player) != mattock {
		t.Fatal("the hotbar didn't follow the moved items")
	}
}

func TestMovingAStackOntoAnotherFillsItAndLeavesTheRest(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	onto, moved := putStack(t, x, player, 10, 70), putStack(t, x, player, 11, 50)

	act(t, x, player, ecs.InventoryAction_Move, 11, 10)

	if countOf(x, onto) != 100 || countOf(x, moved) != 20 || slotItem(inv, 11) != moved {
		t.Fatalf("stacks of %d and %d, want 100 and the 20 left behind in its slot", countOf(x, onto), countOf(x, moved))
	}
}

func TestAStackMovedWhollyOntoAnotherHandsItsHotbarSlotOver(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	moved, onto := putStack(t, x, player, 10, 30), putStack(t, x, player, 11, 40)
	if err := x.BindHotbar(player, 7, moved); err != nil {
		t.Fatal(err)
	}
	if err := x.SelectHotbarSlot(player, 7); err != nil {
		t.Fatal(err)
	}

	act(t, x, player, ecs.InventoryAction_Move, 10, 11)

	if x.HasEntity(moved) || slotItem(inv, 10) != -1 || countOf(x, onto) != 70 {
		t.Fatal("the emptied stack wasn't taken out and removed")
	}
	if hotbarItem(inv, 7) != onto || x.IsDisabled(onto) {
		t.Fatal("the hotbar slot and the selection didn't go over to the stack the wood went into")
	}
}

func TestAStackMovedOntoABoundOneLeavesItsHotbarSlotEmpty(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	moved, onto := putStack(t, x, player, 10, 30), putStack(t, x, player, 11, 40)
	if err := x.BindHotbar(player, 7, moved); err != nil {
		t.Fatal(err)
	}
	if err := x.BindHotbar(player, 8, onto); err != nil {
		t.Fatal(err)
	}

	act(t, x, player, ecs.InventoryAction_Move, 10, 11)

	if hotbarItem(inv, 7) != -1 || hotbarItem(inv, 8) != onto {
		t.Fatal("the stack ended up on two hotbar slots, or lost its own")
	}
}

func TestBindingPointsTheHotbarSlotAtTheItemAndOnlyThatOne(t *testing.T) {
	x, player, mattock, _, gun := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)

	act(t, x, player, ecs.InventoryAction_Bind, 2, 0)
	if hotbarItem(inv, 0) != gun || hotbarItem(inv, 2) != -1 {
		t.Fatal("the gun wasn't moved onto the first hotbar slot")
	}
	if inHand(x, player) != gun || x.Disabled.HasComponent(gun) || !x.Disabled.HasComponent(mattock) {
		t.Fatal("binding the selected hotbar slot didn't put the gun in hand")
	}
	if slotItem(inv, 0) != mattock || inv.HotbarSlotOf(mattock) != -1 {
		t.Fatal("the mattock should still be in its slot, on no hotbar slot")
	}

	act(t, x, player, ecs.InventoryAction_Unbind, 0, 0)
	if hotbarItem(inv, 0) != -1 || inHand(x, player) != -1 || slotItem(inv, 2) != gun {
		t.Fatal("unbinding didn't empty the hand while keeping the gun in the inventory")
	}
}

func TestBindingAnEmptySlotDoesNothing(t *testing.T) {
	x, player, mattock, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)

	act(t, x, player, ecs.InventoryAction_Bind, 30, 0)

	if hotbarItem(inv, 0) != mattock {
		t.Fatal("binding an empty slot changed the hotbar")
	}
}

func TestDroppingTheLastOfAnItemEmptiesItsHotbarSlot(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	press(t, x, player, ecs.InputState{SelectSlot: 3})
	press(t, x, player, ecs.InputState{DropItem: true})
	testutil.RemoveScheduled(t, x)

	if hotbarItem(inv, 2) != -1 || slotItem(inv, 2) != -1 {
		t.Fatal("the dropped gun is still on the hotbar or in the inventory")
	}
	checkConsistency(t, x)
}

func TestNewItemsPickedUpGoOnTheFirstFreeHotbarSlot(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)

	pickUpWood(t, x, player)
	wood := slotItem(inv, 5)
	if hotbarItem(inv, 5) != wood {
		t.Fatal("the wood didn't go onto the first free hotbar slot")
	}

	if err := x.BindHotbar(player, 5, -1); err != nil {
		t.Fatal(err)
	}
	pickUpWood(t, x, player)
	if hotbarItem(inv, 5) != -1 || countOf(x, wood) != 2 {
		t.Fatal("topping up a stack put it back on the hotbar")
	}
	checkConsistency(t, x)
}

func TestNewItemsPickedUpWithAFullHotbarStayOffIt(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	for h := 5; h < inv.HotbarSize(); h++ {
		filler := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewItemComponent())
		if err := x.PutInInventory(player, h, filler); err != nil {
			t.Fatal(err)
		}
		if err := x.BindHotbar(player, h, filler); err != nil {
			t.Fatal(err)
		}
	}

	pickUpWood(t, x, player)

	if wood := slotItem(inv, inv.HotbarSize()); wood == -1 || inv.HotbarSlotOf(wood) != -1 {
		t.Fatal("the wood weren't picked up off the hotbar")
	}
	checkConsistency(t, x)
}
