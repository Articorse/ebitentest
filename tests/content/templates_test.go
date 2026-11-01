package content_test

import (
	"ebittest/content/templates"
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
	if len(members) != 2 {
		t.Fatalf("player stored with %d entities, want itself and its equipment", len(members))
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
