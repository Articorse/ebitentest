package ecs_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestCompositionMembers(t *testing.T) {
	x := testutil.NewECS(t)
	crab := testutil.AddMovable(t, x, utils.Vec2f{}, true)
	claw := testutil.AddMovable(t, x, utils.Vec2f{}, true)
	fire := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	item := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	rider := testutil.AddMovable(t, x, utils.Vec2f{}, true)
	testutil.Attach(t, x, claw, crab, ecs.Link_Part)
	testutil.Attach(t, x, fire, crab, ecs.Link_Part)
	testutil.Attach(t, x, item, claw, ecs.Link_Held)
	testutil.Attach(t, x, rider, crab, ecs.Link_Attached)

	members, err := x.GetCompositionComponents(crab)
	if err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]struct {
		e      common.EntityId
		stored bool
	}{
		"root":                {crab, true},
		"persistent part":     {claw, true},
		"non-persistent part": {fire, false},
		"held item":           {item, true},
		"attached child":      {rider, false},
	} {
		if _, stored := members[want.e]; stored != want.stored {
			t.Errorf("%s: stored %v, want %v", name, stored, want.stored)
		}
	}

	y := testutil.Restore(t, members)
	if err := y.CheckConsistency(); err != nil {
		t.Fatalf("restored composition is inconsistent: %v", err)
	}
	if children := y.HierarchyManager.GetChildren(crab, y); len(children) != 1 || children[0] != claw {
		t.Fatalf("restored crab's children %v, want [%d]", children, claw)
	}
}

func TestAttachedRootStoredWithWorldTransform(t *testing.T) {
	x := testutil.NewECS(t)
	platform := testutil.AddMovable(t, x, utils.Vec2f{X: 80, Y: 100}, false)
	rider := testutil.AddMovable(t, x, utils.Vec2f{X: 90, Y: 100}, true)
	testutil.Attach(t, x, rider, platform, ecs.Link_Attached)
	testutil.SetPos(t, x, platform, utils.Vec2f{X: 100, Y: 100})
	want := testutil.WorldPos(t, x, rider)

	members, err := x.GetCompositionComponents(rider)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := members[platform]; ok {
		t.Fatal("the platform the rider is attached to was stored with it")
	}

	y := testutil.Restore(t, members)
	if parent := y.HierarchyManager.GetParent(rider, y); parent != -1 {
		t.Fatalf("restored rider has parent %d", parent)
	}
	if got := testutil.WorldPos(t, y, rider); !testutil.ApproxEqual(got, want) {
		t.Fatalf("restored rider at %v, want %v", got, want)
	}
}
