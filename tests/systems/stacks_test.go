package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

// In whole kilograms, of the wood these tests use
func countOf(x *ecs.ECSContainer, item common.EntityId) int {
	itemComp, ok := x.Items.Get(item)
	if !ok {
		return 0
	}
	return itemComp.Amount() / data.MilligramsPerKg
}

// Spawns a kilogram of wood where the player stands and gives it to the player
func pickUpWood(t *testing.T, x *ecs.ECSContainer, player common.EntityId) bool {
	t.Helper()
	wood, err := x.SpawnItem(data.Item_Wood, testutil.WorldPos(t, x, player))
	if err != nil {
		t.Fatal(err)
	}
	if err := x.GiveItem(player, wood); err != nil {
		t.Fatal(err)
	}
	given := x.Parent(wood) == player || !x.HasEntity(wood) || countOf(x, wood) == 0
	testutil.RemoveScheduled(t, x)
	return given
}

func TestWoodStacksUpToAHundredPerSlot(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	for range 101 {
		if !pickUpWood(t, x, player) {
			t.Fatal("a wood wasn't picked up into a hotbar with room")
		}
	}

	inv, _ := x.Inventories.Get(player)
	first, _ := inv.Slot(5)
	second, _ := inv.Slot(6)
	if countOf(x, first) != 100 || countOf(x, second) != 1 {
		t.Fatalf("101 wood went into stacks of %d and %d, want 100 and 1", countOf(x, first), countOf(x, second))
	}
	if n := len(x.Items.GetEntities()); n != 8 {
		t.Fatalf("%d items exist, want the 4 tools, the torches, the spacesuit and 2 stacks: the wood used to top up a stack should be gone", n)
	}
	checkConsistency(t, x)
}

func fillFreeSlots(t *testing.T, x *ecs.ECSContainer, player common.EntityId) {
	t.Helper()
	inv, _ := x.Inventories.Get(player)
	for slot := inv.FirstFreeSlot(); slot != -1; slot = inv.FirstFreeSlot() {
		filler := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewItemComponent())
		if err := x.PutInInventory(player, slot, filler); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAFullHotbarStillTakesWhatFitsOnAStack(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	pickUpWood(t, x, player)
	fillFreeSlots(t, x, player)

	if !pickUpWood(t, x, player) {
		t.Fatal("a wood wasn't taken onto the stack with every slot taken")
	}
	inv, _ := x.Inventories.Get(player)
	stack, _ := inv.Slot(5)
	if countOf(x, stack) != 2 {
		t.Fatalf("the stack has %d wood, want 2", countOf(x, stack))
	}
}

func TestAFullHotbarWithFullStacksLeavesItemsAlone(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	for range 100 {
		pickUpWood(t, x, player)
	}
	fillFreeSlots(t, x, player)

	wood, err := x.SpawnItem(data.Item_Wood, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	if x.RoomFor(player, data.Item_Wood) != 0 {
		t.Fatal("there's room for wood with every slot taken and the stack full")
	}
	if err := x.GiveItem(player, wood); err != nil {
		t.Fatal(err)
	}
	if x.Parent(wood) != -1 || countOf(x, wood) != 1 {
		t.Fatal("with every slot taken and the stack full, the wood was taken")
	}
}

func TestUniqueItemsNeverStack(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	fillFreeSlots(t, x, player)

	loose := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewItemComponent())
	if err := x.GiveItem(player, loose); err != nil {
		t.Fatal(err)
	}
	if x.Parent(loose) != -1 {
		t.Fatal("a unique item was stacked onto another one")
	}
}
