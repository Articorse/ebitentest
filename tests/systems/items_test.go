package systems_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

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

func TestGivenItemsGoIntoTheLowestFreeSlotAndOnlyIntoHandWhenItsSelected(t *testing.T) {
	x, player, _, _, gun := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	free := inv.FirstFreeSlot()
	given := testutil.AddMovable(t, x, utils.Vec2f{X: 500}, false, ecs.NewNamedItemComponent("Thing"))

	if err := x.GiveItem(player, given); err != nil {
		t.Fatal(err)
	}
	if slotOf(x, player, given) != free {
		t.Fatalf("the given item went into slot %d, want %d, the lowest free one", slotOf(x, player, given), free)
	}
	if !x.IsDisabled(given) || inHand(x, player) == given {
		t.Fatal("the item given into a slot that isn't selected is enabled or in hand")
	}
	if pos := testutil.WorldPos(t, x, given); pos != testutil.WorldPos(t, x, player) {
		t.Fatalf("the given item is at %v, want with its owner at %v", pos, testutil.WorldPos(t, x, player))
	}
	if slotOf(x, player, gun) == -1 {
		t.Fatal("giving an item moved another out")
	}
	checkConsistency(t, x)
}

func TestItemsArentGivenToAFullInventory(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	for slot := inv.FirstFreeSlot(); slot != -1; slot = inv.FirstFreeSlot() {
		filler := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewItemComponent())
		if err := x.PutInInventory(player, slot, filler); err != nil {
			t.Fatal(err)
		}
	}
	extra := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewItemComponent())
	if err := x.GiveItem(player, extra); err != nil {
		t.Fatal(err)
	}
	if x.Parent(extra) != -1 {
		t.Fatal("an item was given to a full inventory")
	}
}
