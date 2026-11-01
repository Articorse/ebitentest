package ecs_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

func TestStorageIteratesInEntityIdOrder(t *testing.T) {
	x := testutil.NewECS(t)
	a := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))
	b := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewDefaultVelocityComponent())
	c := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewDefaultVelocityComponent())

	x.AddComponent(a, ecs.NewDefaultVelocityComponent())

	if got, want := x.Velocities.GetEntities(), []common.EntityId{a, b, c}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestAddingComponentLeavesSliceBeingIteratedUnchanged(t *testing.T) {
	x := testutil.NewECS(t)
	a := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))
	testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewDefaultVelocityComponent())
	testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewDefaultVelocityComponent())

	iterating := x.Velocities.GetEntities()
	before := slices.Clone(iterating)
	x.AddComponent(a, ecs.NewDefaultVelocityComponent())

	if !slices.Equal(iterating, before) {
		t.Fatalf("slice being iterated changed from %v to %v", before, iterating)
	}
}

func TestRemovedEntityCanBeAddedBackInOrder(t *testing.T) {
	x := testutil.NewECS(t)
	a := testutil.Add(t, x, ecs.NewDefaultVelocityComponent())
	b := testutil.Add(t, x, ecs.NewDefaultVelocityComponent())
	c := testutil.Add(t, x, ecs.NewDefaultVelocityComponent())

	x.ScheduleRemoveEntity(b)
	testutil.RemoveScheduled(t, x)
	if got, want := x.Velocities.GetEntities(), []common.EntityId{a, c}; !slices.Equal(got, want) {
		t.Fatalf("after removal got %v, want %v", got, want)
	}

	if err := x.AddEntityWithId(b, ecs.NewDefaultVelocityComponent()); err != nil {
		t.Fatal(err)
	}
	if got, want := x.Velocities.GetEntities(), []common.EntityId{a, b, c}; !slices.Equal(got, want) {
		t.Fatalf("after re-adding got %v, want %v", got, want)
	}
}

func TestAddingComponentTwiceKeepsOneEntry(t *testing.T) {
	x := testutil.NewECS(t)
	e := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))
	x.AddComponent(e, ecs.NewTransformComponent(utils.Vec2f{X: 1}, 1, 0))

	if got, want := x.Transforms.GetEntities(), []common.EntityId{e}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
