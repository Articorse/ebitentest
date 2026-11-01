package ecs_test

import (
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestRemovedEntityLeavesTheRestOfItsGridCell(t *testing.T) {
	x := testutil.NewECS(t)
	gone := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	stays := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	cell := utils.Vec2i{}
	x.TickState.CollisionGrid[cell] = []common.EntityId{gone, stays}

	x.ScheduleRemoveEntity(gone)
	testutil.RemoveScheduled(t, x)

	if got := x.TickState.CollisionGrid[cell]; len(got) != 1 || got[0] != stays {
		t.Fatalf("cell holds %v, want only %d", got, stays)
	}
}
