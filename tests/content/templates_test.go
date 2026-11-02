package content_test

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestPlayerTemplateWithStartingEquipment(t *testing.T) {
	x := testutil.NewECS(t)
	p, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	if err := templates.SpawnStartingEquipment(x, p); err != nil {
		t.Fatal(err)
	}

	if err := x.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
	if !x.ChunkLoaders.HasComponent(p) {
		t.Fatal("player isn't a chunkloader")
	}

	members, err := x.GetCompositionComponents(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Fatalf("player stored with %d entities, want itself, the bazooka and the gun", len(members))
	}
}

func TestStartingEquipmentIsHeldWherePlayerIs(t *testing.T) {
	x := testutil.NewECS(t)
	p, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	spawn := utils.Vec2f{X: 48, Y: -32}
	testutil.SetPos(t, x, p, spawn)
	if err := templates.SpawnStartingEquipment(x, p); err != nil {
		t.Fatal(err)
	}

	equipper, _ := x.Equippers.Get(p)
	for _, e := range equipper.Equipment() {
		if got := testutil.WorldPos(t, x, e); !testutil.ApproxEqual(got, spawn) {
			t.Fatalf("equipment %d is at %v, want the player's position %v", e, got, spawn)
		}
	}
}

func TestPlayerStartsWithTheBazookaSelected(t *testing.T) {
	x := testutil.NewECS(t)
	p, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	if err := templates.SpawnStartingEquipment(x, p); err != nil {
		t.Fatal(err)
	}

	inv, ok := x.Inventories.Get(p)
	if !ok || inv.Capacity() != data.HotbarSlots {
		t.Fatalf("player inventory %v with %d slots, want %d", ok, inv.Capacity(), data.HotbarSlots)
	}
	equipper, _ := x.Equippers.Get(p)
	bazooka, _ := equipper.InSlot(ecs.Equip_MainHand)
	if item, ok := inv.Slot(inv.Selected()); !ok || item != bazooka {
		t.Fatalf("the selected slot holds %d (%v), want the bazooka %d in hand", item, ok, bazooka)
	}
}
