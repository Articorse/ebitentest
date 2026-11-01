package ecs_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"slices"
	"testing"
)

func TestAttachRefusesCycles(t *testing.T) {
	x := testutil.NewECS(t)
	a := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	b := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	c := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	testutil.Attach(t, x, b, a, ecs.Link_Part)
	testutil.Attach(t, x, c, b, ecs.Link_Part)

	if err := x.Attach(a, a, ecs.Link_Part); err == nil {
		t.Fatal("attaching an entity to itself was allowed")
	}
	if err := x.Attach(a, c, ecs.Link_Part); err == nil {
		t.Fatal("attaching an entity to its grandchild was allowed")
	}
	if x.Parent(a) != -1 || x.Parent(c) != b {
		t.Fatalf("refused attach changed links: parent of a %d, parent of c %d", x.Parent(a), x.Parent(c))
	}
}

func TestReattachingKeepsOneChildEntry(t *testing.T) {
	x := testutil.NewECS(t)
	p := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	c := testutil.AddMovable(t, x, utils.Vec2f{X: 5}, false)

	testutil.Attach(t, x, c, p, ecs.Link_Held)
	if err := x.Detach(c); err != nil {
		t.Fatal(err)
	}
	if children := x.Children(p); len(children) != 0 || x.Parent(c) != -1 {
		t.Fatalf("after detach: children %v, parent %d", children, x.Parent(c))
	}

	testutil.Attach(t, x, c, p, ecs.Link_Held)
	testutil.Attach(t, x, c, p, ecs.Link_Held)
	if children := x.Children(p); !slices.Equal(children, []common.EntityId{c}) {
		t.Fatalf("children %v, want [%d]", children, c)
	}

	if dfs := x.ChildrenDFS(p); !slices.Equal(dfs, []common.EntityId{c}) {
		t.Fatalf("ChildrenDFS %v, want [%d]", dfs, c)
	}
}

func TestChildrenHeldByCallerUnaffectedByLaterChanges(t *testing.T) {
	x := testutil.NewECS(t)
	p := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	a := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	b := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	testutil.Attach(t, x, a, p, ecs.Link_Held)
	testutil.Attach(t, x, b, p, ecs.Link_Held)

	children := x.Children(p)
	if err := x.Detach(a); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(children, []common.EntityId{a, b}) {
		t.Fatalf("children held by the caller changed to %v", children)
	}

	children[0] = p
	if got := x.Children(p); !slices.Equal(got, []common.EntityId{b}) {
		t.Fatalf("changing the returned children changed the hierarchy to %v", got)
	}
}

func TestRemovingChildUnlinksItFromParent(t *testing.T) {
	x := testutil.NewECS(t)
	p := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	c := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	testutil.Attach(t, x, c, p, ecs.Link_Part)

	x.ScheduleRemoveEntity(c)
	testutil.RemoveScheduled(t, x)

	if children := x.Children(p); len(children) != 0 {
		t.Fatalf("parent still lists children %v", children)
	}
}

func TestKillingByLinkKind(t *testing.T) {
	x := testutil.NewECS(t)
	crab := testutil.AddMovable(t, x, utils.Vec2f{X: 10, Y: 10}, true)
	claw := testutil.AddMovable(t, x, utils.Vec2f{X: 15, Y: 10}, true)
	item := testutil.AddMovable(t, x, utils.Vec2f{X: 10, Y: 20}, true)
	rider := testutil.AddMovable(t, x, utils.Vec2f{X: 10, Y: 5}, true)
	testutil.Attach(t, x, claw, crab, ecs.Link_Part)
	testutil.Attach(t, x, item, crab, ecs.Link_Held)
	testutil.Attach(t, x, rider, crab, ecs.Link_Attached)
	testutil.SetPos(t, x, crab, utils.Vec2f{X: 40, Y: 40})
	itemPos := testutil.WorldPos(t, x, item)
	riderPos := testutil.WorldPos(t, x, rider)

	x.ScheduleKillEntity(crab)
	testutil.RemoveScheduled(t, x)

	if x.HasEntity(crab) || x.HasEntity(claw) {
		t.Fatal("killed entity or its part survived")
	}
	if !x.HasEntity(item) || x.Parent(item) != -1 {
		t.Fatalf("held item not dropped: alive %v, parent %d", x.HasEntity(item), x.Parent(item))
	}
	if !x.HasEntity(rider) || x.Parent(rider) != -1 {
		t.Fatalf("attached rider not detached: alive %v, parent %d", x.HasEntity(rider), x.Parent(rider))
	}
	if got := testutil.WorldPos(t, x, item); !testutil.ApproxEqual(got, itemPos) {
		t.Fatalf("dropped item moved from %v to %v", itemPos, got)
	}
	if got := testutil.WorldPos(t, x, rider); !testutil.ApproxEqual(got, riderPos) {
		t.Fatalf("detached rider moved from %v to %v", riderPos, got)
	}
}

func TestKillingDropsItemsHeldByParts(t *testing.T) {
	x := testutil.NewECS(t)
	crab := testutil.AddMovable(t, x, utils.Vec2f{X: 10, Y: 10}, true)
	claw := testutil.AddMovable(t, x, utils.Vec2f{X: 15, Y: 10}, true)
	item := testutil.AddMovable(t, x, utils.Vec2f{X: 20, Y: 10}, true)
	testutil.Attach(t, x, claw, crab, ecs.Link_Part)
	testutil.Attach(t, x, item, claw, ecs.Link_Held)
	testutil.SetPos(t, x, crab, utils.Vec2f{X: 40, Y: 40})
	itemPos := testutil.WorldPos(t, x, item)

	x.ScheduleKillEntity(crab)
	testutil.RemoveScheduled(t, x)

	if x.HasEntity(claw) {
		t.Fatal("part survived its parent being killed")
	}
	if !x.HasEntity(item) || x.Parent(item) != -1 {
		t.Fatalf("item held by the part not dropped: alive %v, parent %d", x.HasEntity(item), x.Parent(item))
	}
	if got := testutil.WorldPos(t, x, item); !testutil.ApproxEqual(got, itemPos) {
		t.Fatalf("dropped item moved from %v to %v", itemPos, got)
	}
	if err := x.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestKillingPartLeavesParent(t *testing.T) {
	x := testutil.NewECS(t)
	crab := testutil.AddMovable(t, x, utils.Vec2f{}, true)
	claw := testutil.AddMovable(t, x, utils.Vec2f{}, true)
	testutil.Attach(t, x, claw, crab, ecs.Link_Part)

	x.ScheduleKillEntity(claw)
	testutil.RemoveScheduled(t, x)

	if x.HasEntity(claw) || !x.HasEntity(crab) {
		t.Fatalf("claw alive %v, crab alive %v", x.HasEntity(claw), x.HasEntity(crab))
	}
	if children := x.Children(crab); len(children) != 0 {
		t.Fatalf("crab still lists children %v", children)
	}
}

func TestRemovingHolderRemovesHeldItem(t *testing.T) {
	x := testutil.NewECS(t)
	holder := testutil.AddMovable(t, x, utils.Vec2f{}, true)
	held := testutil.AddMovable(t, x, utils.Vec2f{}, true)
	testutil.Attach(t, x, held, holder, ecs.Link_Held)

	x.ScheduleRemoveEntity(holder)
	testutil.RemoveScheduled(t, x)

	if x.HasEntity(held) {
		t.Fatal("held item survived its holder being removed")
	}
}

func TestAttachAndDetachKeepWorldTransform(t *testing.T) {
	x := testutil.NewECS(t)
	p := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{X: 50, Y: -20}, 1, 0.7), ecs.NewVelocityComponentWithParams(utils.Vec2f{X: 1, Y: 2}, 1, 1))
	c := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{X: 10, Y: 30}, 1, 0.3), ecs.NewVelocityComponentWithParams(utils.Vec2f{}, 1, 1))

	testutil.Attach(t, x, c, p, ecs.Link_Part)
	assertWorldTransform(t, x, c, utils.Vec2f{X: 10, Y: 30}, 0.3)

	if err := x.SetWorldPos(c, utils.Vec2f{X: -7, Y: 4}); err != nil {
		t.Fatal(err)
	}
	assertWorldTransform(t, x, c, utils.Vec2f{X: -7, Y: 4}, 0.3)

	if err := x.SetWorldVector(c, utils.Vec2f{X: 3, Y: -1}); err != nil {
		t.Fatal(err)
	}
	v, err := x.WorldVector(c)
	if err != nil {
		t.Fatal(err)
	}
	if !testutil.ApproxEqual(v, utils.Vec2f{X: 3, Y: -1}) {
		t.Fatalf("world velocity %v, want {3 -1}", v)
	}

	if err := x.Detach(c); err != nil {
		t.Fatal(err)
	}
	assertWorldTransform(t, x, c, utils.Vec2f{X: -7, Y: 4}, 0.3)
}

func assertWorldTransform(t *testing.T, x *ecs.ECSContainer, e common.EntityId, wantPos utils.Vec2f, wantRot float64) {
	t.Helper()

	pos := testutil.WorldPos(t, x, e)
	rot, err := x.WorldRotation(e)
	if err != nil {
		t.Fatal(err)
	}
	if !testutil.ApproxEqual(pos, wantPos) || math.Abs(rot-wantRot) > 1e-9 {
		t.Fatalf("world transform %v %v, want %v %v", pos, rot, wantPos, wantRot)
	}
}
