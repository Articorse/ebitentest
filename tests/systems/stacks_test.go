package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func countOf(x *ecs.ECSContainer, item common.EntityId) int {
	itemComp, ok := x.Items.Get(item)
	if !ok {
		return 0
	}
	return itemComp.Count()
}

// Spawns a plank where the player stands and has the player pick it up
func pickUpPlank(t *testing.T, x *ecs.ECSContainer, player common.EntityId) bool {
	t.Helper()
	plank, err := x.SpawnItem(data.Item_Planks, testutil.WorldPos(t, x, player))
	if err != nil {
		t.Fatal(err)
	}
	picked, err := x.PickUpItem(player, plank)
	if err != nil {
		t.Fatal(err)
	}
	testutil.RemoveScheduled(t, x)
	return picked
}

func TestPlanksStackUpToAHundredPerSlot(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	for range 101 {
		if !pickUpPlank(t, x, player) {
			t.Fatal("a plank wasn't picked up into a hotbar with room")
		}
	}

	inv, _ := x.Inventories.Get(player)
	first, _ := inv.Slot(4)
	second, _ := inv.Slot(5)
	if countOf(x, first) != 100 || countOf(x, second) != 1 {
		t.Fatalf("101 planks went into stacks of %d and %d, want 100 and 1", countOf(x, first), countOf(x, second))
	}
	if n := len(x.Items.GetEntities()); n != 6 {
		t.Fatalf("%d items exist, want the 4 tools and 2 stacks: the planks used to top up a stack should be gone", n)
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
	pickUpPlank(t, x, player)
	fillFreeSlots(t, x, player)

	plank, err := x.SpawnItem(data.Item_Planks, utils.Vec2f{X: data.ItemPullReachPx * 0.8})
	if err != nil {
		t.Fatal(err)
	}
	tickItems(x, 500)
	testutil.RemoveScheduled(t, x)

	inv, _ := x.Inventories.Get(player)
	stack, _ := inv.Slot(4)
	if countOf(x, stack) != 2 || x.HasEntity(plank) {
		t.Fatalf("the stack has %d planks and the plank on the ground still exists %v; want it pulled in and stacked",
			countOf(x, stack), x.HasEntity(plank))
	}
}

func TestAFullHotbarWithFullStacksLeavesItemsAlone(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	for range 100 {
		pickUpPlank(t, x, player)
	}
	fillFreeSlots(t, x, player)

	start := utils.Vec2f{X: data.ItemPullReachPx * 0.8}
	plank, err := x.SpawnItem(data.Item_Planks, start)
	if err != nil {
		t.Fatal(err)
	}
	tickItems(x, 500)

	if x.Parent(plank) != -1 || testutil.WorldPos(t, x, plank) != start {
		t.Fatalf("with every slot taken and the stack full, the plank was pulled to %v or picked up", testutil.WorldPos(t, x, plank))
	}
}

func TestDroppingFromAStackDropsOneAtATime(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	for range 3 {
		pickUpPlank(t, x, player)
	}
	inv, _ := x.Inventories.Get(player)
	stack, _ := inv.Slot(4)
	press(t, x, player, ecs.InputState{SelectSlot: 5})

	aim := utils.Vec2f{X: 100}
	for _, want := range []int{2, 1} {
		before := len(x.Items.GetEntities())
		press(t, x, player, ecs.InputState{DropItem: true, AimPos: aim, HasAim: true})

		dropped := x.Items.GetEntities()[before:]
		if countOf(x, stack) != want || len(dropped) != 1 || countOf(x, dropped[0]) != 1 {
			t.Fatalf("after dropping, the stack has %d and %v were dropped; want %d left and a single plank dropped",
				countOf(x, stack), dropped, want)
		}
		if !x.Tosses.HasComponent(dropped[0]) {
			t.Fatal("the plank dropped from the stack wasn't tossed")
		}
	}

	press(t, x, player, ecs.InputState{DropItem: true, AimPos: aim, HasAim: true})
	if x.Parent(stack) != -1 || countOf(x, stack) != 1 {
		t.Fatalf("dropping the last plank left %d in the stack, held by %d; want the stack itself dropped", countOf(x, stack), x.Parent(stack))
	}
	checkConsistency(t, x)
}

func TestItemsArentPickedUpWhileFlying(t *testing.T) {
	x, player, mattock, _, _ := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{DropItem: true, AimPos: utils.Vec2f{X: 100}, HasAim: true})
	// Its delay runs out without the toss moving on, which only the item system does
	itemComp, _ := x.Items.Get(mattock)
	for range data.ItemPickupDelayMs / data.TickMs {
		itemComp.TickPickupDelay()
	}
	if !itemComp.CanBePickedUp() {
		t.Fatal("the pickup delay didn't run out")
	}
	testutil.SetPos(t, x, mattock, utils.Vec2f{})

	if x.CanPickUp(player, mattock) {
		t.Fatal("an item can be picked up while it's still tossed through the air")
	}
}

func TestUniqueItemsNeverStack(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	fillFreeSlots(t, x, player)

	loose := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewItemComponent())
	if x.CanPickUp(player, loose) {
		t.Fatal("a unique item can be stacked onto another one")
	}
}
