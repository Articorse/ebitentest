package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/itemsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func tickItems(x *ecs.ECSContainer, ms int) {
	for range ms / data.TickMs {
		itemsystem.Tick(x)
	}
}

func slotOf(x *ecs.ECSContainer, owner, item common.EntityId) int {
	inv, _ := x.Inventories.Get(owner)
	for i := range inv.Capacity() {
		if e, ok := inv.Slot(i); ok && e == item {
			return i
		}
	}
	return -1
}

func checkConsistency(t *testing.T, x *ecs.ECSContainer) {
	t.Helper()
	if err := x.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestDroppingPutsTheSelectedItemUprightWhereItsOwnerStands(t *testing.T) {
	x, player, _, bazooka, _ := newArmedPlayer(t)
	stand := utils.Vec2f{X: 40, Y: -24}
	testutil.SetPos(t, x, player, stand)
	press(t, x, player, ecs.InputState{SelectSlot: 2, AimPos: utils.Vec2f{Y: 100}, HasAim: true})
	for range 30 {
		press(t, x, player, ecs.InputState{AimPos: utils.Vec2f{Y: 100}, HasAim: true})
	}

	press(t, x, player, ecs.InputState{DropItem: true, AimPos: utils.Vec2f{X: -100}, HasAim: true})

	if x.Parent(bazooka) != -1 || slotOf(x, player, bazooka) != -1 || inHand(x, player) != -1 {
		t.Fatalf("after dropping, the bazooka has parent %d, is in slot %d and %d is in hand; want it on its own",
			x.Parent(bazooka), slotOf(x, player, bazooka), inHand(x, player))
	}
	if x.IsDisabled(bazooka) {
		t.Fatal("the dropped bazooka is disabled, so it isn't drawn")
	}
	if pos := testutil.WorldPos(t, x, bazooka); pos != stand {
		t.Fatalf("the bazooka was dropped at %v, want where its owner stands, %v", pos, stand)
	}
	if r, _ := x.WorldRotation(bazooka); r != 0 {
		t.Fatalf("the dropped bazooka lies at %v rad, want upright", r)
	}

	press(t, x, player, ecs.InputState{AimPos: utils.Vec2f{X: -100}, HasAim: true})
	if r, _ := x.WorldRotation(bazooka); r != 0 {
		t.Fatalf("the dropped bazooka turned to %v rad towards its old owner's aim", r)
	}
	checkConsistency(t, x)
}

func TestDroppingAnEmptySlotDoesNothing(t *testing.T) {
	x, player, mattock, bazooka, gun := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{SelectSlot: 6, DropItem: true})

	for _, item := range []common.EntityId{mattock, bazooka, gun} {
		if x.Parent(item) != player {
			t.Fatalf("item %d was dropped from an empty slot", item)
		}
	}
	checkConsistency(t, x)
}

func TestDroppedItemsArePickedBackUpOnlyAfterTheirDelay(t *testing.T) {
	x, player, mattock, _, _ := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{DropItem: true})
	// Within reach, but away from the player, so that picking it up has to move it into the player's hand
	testutil.SetPos(t, x, mattock, utils.Vec2f{X: data.ItemPickupReachPx / 2, Y: -data.ItemPickupReachPx / 3})

	tickItems(x, data.ItemPickupDelayMs-data.TickMs)
	if x.Parent(mattock) != -1 {
		t.Fatal("the mattock was picked back up before its delay was over")
	}

	tickItems(x, 2*data.TickMs)
	if slotOf(x, player, mattock) != 0 || inHand(x, player) != mattock || x.IsDisabled(mattock) {
		t.Fatalf("after its delay, the mattock is in slot %d, %d is in hand and it's disabled %v; want it back in the selected slot, in hand",
			slotOf(x, player, mattock), inHand(x, player), x.IsDisabled(mattock))
	}
	if pos := testutil.WorldPos(t, x, mattock); pos != testutil.WorldPos(t, x, player) {
		t.Fatalf("the picked up mattock is at %v, want in its owner's hand at %v", pos, testutil.WorldPos(t, x, player))
	}
	checkConsistency(t, x)
}

func TestPickedUpItemsGoIntoTheLowestFreeSlot(t *testing.T) {
	x, player, mattock, _, gun := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{DropItem: true})
	press(t, x, player, ecs.InputState{SelectSlot: 3, DropItem: true})
	testutil.SetPos(t, x, mattock, utils.Vec2f{X: 500})

	tickItems(x, data.ItemPickupDelayMs+data.TickMs)

	if slotOf(x, player, gun) != 0 {
		t.Fatalf("the gun went into slot %d, want 0, the lowest free one", slotOf(x, player, gun))
	}
	if !x.IsDisabled(gun) || inHand(x, player) != -1 {
		t.Fatal("the gun was picked up into a slot that isn't selected, but it's enabled or in hand")
	}
	if x.Parent(mattock) != -1 {
		t.Fatal("the mattock was picked up from far away")
	}
	checkConsistency(t, x)
}

func TestItemsArentPickedUpIntoAFullInventory(t *testing.T) {
	x, player, mattock, _, _ := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{DropItem: true})
	inv, _ := x.Inventories.Get(player)
	for slot := inv.FirstFreeSlot(); slot != -1; slot = inv.FirstFreeSlot() {
		filler := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewItemComponent())
		if err := x.PutInInventory(player, slot, filler); err != nil {
			t.Fatal(err)
		}
	}

	tickItems(x, data.ItemPickupDelayMs+data.TickMs)
	if x.Parent(mattock) != -1 {
		t.Fatal("an item was picked up into a full inventory")
	}
}
