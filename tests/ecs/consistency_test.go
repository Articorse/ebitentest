package ecs_test

import (
	"ebittest/content/templates"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"strings"
	"testing"
)

func TestConsistencyCheck(t *testing.T) {
	x := testutil.NewECS(t)
	p, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	if err := templates.SpawnStartingEquipment(x, p); err != nil {
		t.Fatal(err)
	}
	if err := x.CheckConsistency(); err != nil {
		t.Fatalf("fresh player reported inconsistent: %v", err)
	}

	children := x.HierarchyManager.GetChildren(p, x)
	if len(children) != 1 {
		t.Fatalf("player children %v, want the starting equipment", children)
	}
	equipment := children[0]
	if err := x.HierarchyManager.Detach(equipment, x); err != nil {
		t.Fatal(err)
	}
	testutil.AddMovable(t, x, utils.Vec2f{}, true, ecs.NewTetherComponent(9999))
	testutil.AddMovable(t, x, utils.Vec2f{}, true, ecs.NewTetherComponent(equipment))

	err = x.CheckConsistency()
	if err == nil {
		t.Fatal("inconsistencies not reported")
	}
	for _, want := range []string{"isn't held by it", "tethered to missing entity 9999", "isn't a chunkloader"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("report doesn't mention %q:\n%v", want, err)
		}
	}
}
