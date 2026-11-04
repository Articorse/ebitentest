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

func TestSmallWheelMovementsAddUpToSteps(t *testing.T) {
	in := ecs.NewInputComponent(nil, 0, nil, ecs.Facing_None)
	for i, step := range []struct {
		down float64
		want int
	}{{0.4, 0}, {0.4, 0}, {0.4, 1}, {2.1, 2}, {-0.5, 0}, {-0.6, -1}, {0.9, 0}, {0.1, 1}, {0, 0}} {
		in.AddWheel(step.down)
		if got := in.WheelSteps(); got != step.want {
			t.Fatalf("call %d scrolled %d steps, want %d", i, got, step.want)
		}
	}
}

func TestWheelMovementWaitsForTheNextTick(t *testing.T) {
	in := ecs.NewInputComponent(nil, 0, nil, ecs.Facing_None)
	for range 3 {
		in.AddWheel(1)
	}
	if got := in.WheelSteps(); got != 3 {
		t.Fatalf("three frames of scrolling between ticks gave %d steps, want 3", got)
	}
}
