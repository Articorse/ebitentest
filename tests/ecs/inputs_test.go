package ecs_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"testing"
)

func TestOnlyCurrentTickInputsAreAvailable(t *testing.T) {
	x := testutil.NewECS(t)
	e := common.EntityId(1)
	x.SetTickInputs(x.TickIdx, map[common.EntityId]ecs.InputState{e: {Analog1X: 1}})

	input, err := x.GetCurrentTickInputsForEntity(e)
	if err != nil {
		t.Fatal(err)
	}
	if input.Analog1X != 1 {
		t.Fatalf("got input %+v, want the one set for this tick", input)
	}

	x.TickIdx++
	if _, err := x.GetCurrentTickInputs(); err == nil {
		t.Fatal("the previous tick's inputs were returned as the current tick's")
	}
}
