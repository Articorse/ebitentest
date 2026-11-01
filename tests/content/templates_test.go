package content_test

import (
	"ebittest/content/templates"
	"ebittest/tests/testutil"
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
