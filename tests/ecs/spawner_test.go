package ecs_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"fmt"
	"math"
	"testing"
)

func TestPointSpawnerPlacesAtRotatedOffset(t *testing.T) {
	x := testutil.NewECS(t)
	sp, err := ecs.NewSpawnerComponent(utils.Vec2f{X: 13}, ecs.SpawnerType_Point, nil, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{X: 100, Y: 100}, 1, 0.5), sp)
	template := templateString(t, x, s)
	want := utils.Vec2f{X: 100 + 13*math.Cos(0.5), Y: 100 + 13*math.Sin(0.5)}

	for range 2 {
		e, err := x.SpawnerManager.Spawn(s, x)
		if err != nil {
			t.Fatal(err)
		}
		if got := testutil.WorldPos(t, x, e); !testutil.ApproxEqual(got, want) {
			t.Fatalf("spawned at %v, want %v", got, want)
		}
		rot, err := x.TransformManager.GetWorldRotation(e, x)
		if err != nil {
			t.Fatal(err)
		}
		if rot != 0.5 {
			t.Fatalf("spawned with rotation %v, want 0.5", rot)
		}
	}

	if got := templateString(t, x, s); got != template {
		t.Fatalf("spawning changed the spawner's components:\n%s\n%s", template, got)
	}
}

func TestShapeSpawnersSpawnWithinShapeAroundSpawner(t *testing.T) {
	x := testutil.NewECS(t)
	for _, sType := range []ecs.SpawnerType{ecs.SpawnerType_Inside, ecs.SpawnerType_Perimeter} {
		circle, err := shapes.NewCircleShape(10, utils.Vec2f{})
		if err != nil {
			t.Fatal(err)
		}
		sp, err := ecs.NewSpawnerComponent(utils.Vec2f{X: 3}, sType, circle, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))
		if err != nil {
			t.Fatal(err)
		}
		s := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{X: 100, Y: 100}, 1, 0), sp)

		for range 20 {
			e, err := x.SpawnerManager.Spawn(s, x)
			if err != nil {
				t.Fatal(err)
			}
			pos := testutil.WorldPos(t, x, e)
			if d := pos.Subtract(utils.Vec2f{X: 103, Y: 100}).Length(); d > 10+1e-9 {
				t.Fatalf("spawner type %d spawned at %v, %.2f from the shape's centre", sType, pos, d)
			}
		}
	}
}

func TestSpawnerRefusedOutsideSimulatedArea(t *testing.T) {
	x := testutil.NewECS(t)
	x.SetSimulatedPosCheck(func(worldPos utils.Vec2f) bool { return worldPos.X < 1000 })
	sp, err := ecs.NewSpawnerComponent(utils.Vec2f{X: 13}, ecs.SpawnerType_Point, nil, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{X: 990}, 1, 0), sp)

	if _, err := x.SpawnerManager.Spawn(s, x); err == nil {
		t.Fatal("spawn outside the simulated area was allowed")
	}
}

func templateString(t *testing.T, x *ecs.ECSContainer, s common.EntityId) string {
	t.Helper()

	comps, err := x.SpawnerManager.GetComponents(s, x)
	if err != nil {
		t.Fatal(err)
	}

	str := ""
	for _, c := range comps {
		dto, err := ecs.ComponentToDto(c)
		if err != nil {
			t.Fatal(err)
		}
		str += fmt.Sprintf("%#v", dto)
	}

	return str
}
