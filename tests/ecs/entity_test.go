package ecs_test

import (
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestAddEntityWithIdRefusesLiveEntity(t *testing.T) {
	x := testutil.NewECS(t)
	e := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))

	if err := x.AddEntityWithId(e, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0)); err == nil {
		t.Fatal("re-adding a live entity was allowed")
	}
}

func TestAddEntityWithIdMovesNextEntityPastIt(t *testing.T) {
	x := testutil.NewECS(t)
	e := x.NextEntity() + 5

	if err := x.AddEntityWithId(e, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0)); err != nil {
		t.Fatal(err)
	}
	if next := x.NextEntity(); next != e+1 {
		t.Fatalf("next entity is %d, want %d", next, e+1)
	}
}

func TestSpawnOutsideSimulatedAreaRefused(t *testing.T) {
	x := testutil.NewECS(t)
	x.SetSimulatedPosCheck(func(worldPos utils.Vec2f) bool { return worldPos.X < 1000 })

	next := x.NextEntity()
	if _, err := x.AddNewEntity(ecs.NewTransformComponent(utils.Vec2f{X: 5000}, 1, 0)); err == nil {
		t.Fatal("spawn outside the simulated area was allowed")
	}
	if x.NextEntity() != next {
		t.Fatal("refused spawn used up an entity ID")
	}

	if _, err := x.AddNewEntity(ecs.NewTransformComponent(utils.Vec2f{X: 10}, 1, 0)); err != nil {
		t.Fatalf("spawn inside the simulated area refused: %v", err)
	}
	if _, err := x.AddNewEntity(ecs.NewHierarchyComponent()); err != nil {
		t.Fatalf("spawn without a transform refused: %v", err)
	}
	if _, err := x.AddNewEntity(ecs.NewTransformComponent(utils.Vec2f{X: 5000}, 1, 0), ecs.NewChunkLoaderComponent(1)); err != nil {
		t.Fatalf("chunkloader spawn refused: %v", err)
	}
}
