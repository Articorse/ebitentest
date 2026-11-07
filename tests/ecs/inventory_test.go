package ecs_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func newOwner(t *testing.T, x *ecs.ECSContainer) common.EntityId {
	t.Helper()
	return testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewInventoryComponent(3, 3, 1000))
}

func newItem(t *testing.T, x *ecs.ECSContainer) common.EntityId {
	t.Helper()
	return testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewItemComponent())
}

func checkConsistency(t *testing.T, x *ecs.ECSContainer) {
	t.Helper()
	if err := x.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestItemPutInAnInventoryIsHeldByItsOwner(t *testing.T) {
	x := testutil.NewECS(t)
	owner, item := newOwner(t, x), newItem(t, x)

	if err := x.PutInInventory(owner, 2, item); err != nil {
		t.Fatal(err)
	}

	inv, _ := x.Inventories.Get(owner)
	if got, ok := inv.Slot(2); !ok || got != item {
		t.Fatalf("slot 2 holds %d (%v), want item %d", got, ok, item)
	}
	if _, ok := inv.Slot(0); ok {
		t.Fatal("an untouched slot isn't empty")
	}
	if x.Parent(item) != owner || x.LinkKind(item) != ecs.Link_Held {
		t.Fatal("the item isn't held by the inventory's owner")
	}
	checkConsistency(t, x)
}

func TestInventoryRefusesWhatDoesntFit(t *testing.T) {
	x := testutil.NewECS(t)
	owner, item, other := newOwner(t, x), newItem(t, x), newItem(t, x)
	notAnItem := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	if err := x.PutInInventory(owner, 0, item); err != nil {
		t.Fatal(err)
	}

	for name, put := range map[string]func() error{
		"an occupied slot":         func() error { return x.PutInInventory(owner, 0, other) },
		"a slot past its capacity": func() error { return x.PutInInventory(owner, 3, other) },
		"a negative slot":          func() error { return x.PutInInventory(owner, -1, other) },
		"an entity without Item":   func() error { return x.PutInInventory(owner, 1, notAnItem) },
		"the same item twice":      func() error { return x.PutInInventory(owner, 1, item) },
		"an owner without one":     func() error { return x.PutInInventory(other, 0, item) },
	} {
		if err := put(); err == nil {
			t.Errorf("putting into %s was allowed", name)
		}
	}

	if err := x.SelectHotbarSlot(owner, 3); err == nil {
		t.Error("selecting a hotbar slot past its size was allowed")
	}
	checkConsistency(t, x)
}

func TestItemLeavingItsOwnerLeavesTheInventory(t *testing.T) {
	x := testutil.NewECS(t)
	owner, detached, removed := newOwner(t, x), newItem(t, x), newItem(t, x)
	for slot, item := range []common.EntityId{detached, removed} {
		if err := x.PutInInventory(owner, slot, item); err != nil {
			t.Fatal(err)
		}
	}

	if err := x.Detach(detached); err != nil {
		t.Fatal(err)
	}
	x.ScheduleRemoveEntity(removed)
	testutil.RemoveScheduled(t, x)

	inv, _ := x.Inventories.Get(owner)
	for slot := range 2 {
		if item, ok := inv.Slot(slot); ok {
			t.Errorf("slot %d still holds %d after it left the owner", slot, item)
		}
	}
	checkConsistency(t, x)
}

func TestInventorySurvivesSaving(t *testing.T) {
	x := testutil.NewECS(t)
	owner, item := newOwner(t, x), newItem(t, x)
	if err := x.PutInInventory(owner, 1, item); err != nil {
		t.Fatal(err)
	}
	if err := x.BindHotbar(owner, 2, item); err != nil {
		t.Fatal(err)
	}
	if err := x.SelectHotbarSlot(owner, 1); err != nil {
		t.Fatal(err)
	}

	members, err := x.GetCompositionComponents(owner)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)

	restored, ok := y.Inventories.Get(owner)
	if !ok || restored.Capacity() != 3 || restored.Selected() != 1 {
		t.Fatalf("restored inventory %v, capacity %d, selected %d; want capacity 3 with slot 1 selected", ok, restored.Capacity(), restored.Selected())
	}
	if got, ok := restored.Slot(1); !ok || got != item || !y.Items.HasComponent(item) {
		t.Fatal("the restored inventory lost its item")
	}
	if got, ok := restored.HotbarItem(2); !ok || got != item {
		t.Fatal("the restored hotbar doesn't point at the item")
	}
	checkConsistency(t, y)
}
