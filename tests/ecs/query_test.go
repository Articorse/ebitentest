package ecs_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

func TestGetReportsMissingComponent(t *testing.T) {
	x := testutil.NewECS(t)
	e := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))

	if c, ok := x.Transforms.Get(e); !ok || c == nil {
		t.Fatal("present component not found")
	}
	if _, ok := x.Velocities.Get(e); ok {
		t.Fatal("missing component found")
	}
}

func TestQueryYieldsEntitiesWithAllComponentsInIdOrder(t *testing.T) {
	x := testutil.NewECS(t)
	a := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))
	b := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewDefaultVelocityComponent())
	testutil.Add(t, x, ecs.NewDefaultVelocityComponent())
	d := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewDefaultVelocityComponent())
	x.AddComponent(a, ecs.NewDefaultVelocityComponent())

	var got []common.EntityId
	for e, c := range ecs.Query2(&x.Transforms, &x.Velocities) {
		if c.A == nil || c.B == nil {
			t.Fatalf("entity %d yielded without its components", e)
		}
		got = append(got, e)
	}

	if want := []common.EntityId{a, b, d}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestQueryFilters(t *testing.T) {
	x := testutil.NewECS(t)
	plain := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))
	persistent := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewPersistentComponent())
	tethered := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewPersistentComponent(), ecs.NewTetherComponent(plain))
	loader := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewChunkLoaderComponent(1))

	cases := []struct {
		name    string
		filters []ecs.Filter
		want    []common.EntityId
	}{
		{"no filters", nil, []common.EntityId{plain, persistent, tethered, loader}},
		{"with", []ecs.Filter{ecs.With(&x.Persistents)}, []common.EntityId{persistent, tethered}},
		{"without", []ecs.Filter{ecs.Without(&x.Tethers, &x.ChunkLoaders)}, []common.EntityId{plain, persistent}},
		{"with and without", []ecs.Filter{ecs.With(&x.Persistents), ecs.Without(&x.Tethers)}, []common.EntityId{persistent}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []common.EntityId
			for e := range ecs.Query1(&x.Transforms, c.filters...) {
				got = append(got, e)
			}
			if !slices.Equal(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestQueryHandsOutLiveComponents(t *testing.T) {
	x := testutil.NewECS(t)
	e := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewDefaultVelocityComponent(),
		ecs.NewPersistentComponent())

	for qe, c := range ecs.Query4(&x.Hierarchies, &x.Transforms, &x.Velocities, &x.Persistents) {
		if qe != e || c.A == nil || c.B == nil || c.C == nil || c.D == nil {
			t.Fatalf("entity %d yielded with missing components", qe)
		}
		if live, _ := x.Transforms.Get(e); live != c.B {
			t.Fatal("query handed out a copy instead of the live component")
		}
	}
}

func TestQueryStopsOnBreak(t *testing.T) {
	x := testutil.NewECS(t)
	for range 3 {
		testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewDefaultVelocityComponent(), ecs.NewPersistentComponent())
	}

	n := 0
	for range ecs.Query3(&x.Transforms, &x.Velocities, &x.Persistents) {
		n++
		break
	}
	if n != 1 {
		t.Fatalf("loop body ran %d times after break", n)
	}
}
