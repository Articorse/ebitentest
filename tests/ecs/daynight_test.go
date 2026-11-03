package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/systems/visionsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

const cycleTicks = data.DayTicks + data.DuskTicks + data.NightTicks + data.DawnTicks

func TestDarknessFollowsTheDayNightCycle(t *testing.T) {
	dusk := uint64(data.DayTicks)
	night := dusk + data.DuskTicks
	dawn := night + data.NightTicks
	for _, c := range []struct {
		name string
		tick uint64
		want float64
	}{
		{"the start of the first day", 0, 0},
		{"the end of the day", dusk - 1, 0},
		{"the start of dusk", dusk, 0},
		{"halfway through dusk", dusk + data.DuskTicks/2, 0.5},
		{"the start of the night", night, 1},
		{"the end of the night", dawn - 1, 1},
		{"halfway through dawn", dawn + data.DawnTicks/2, 0.5},
		{"a quarter through dusk", dusk + data.DuskTicks/4, 0.25},
		{"a quarter through dawn", dawn + data.DawnTicks/4, 0.75},
		{"the start of the next day", cycleTicks, 0},
		{"halfway through a later dusk", 7*cycleTicks + dusk + data.DuskTicks/2, 0.5},
	} {
		if got := ecs.Darkness(c.tick); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s (tick %d): darkness %v, want %v", c.name, c.tick, got, c.want)
		}
	}
}

func TestTheCycleIsThirtySecondsOfTenDayFiveDuskTenNightFiveDawn(t *testing.T) {
	if data.DayTicks != 10*data.TPS || data.DuskTicks != 5*data.TPS || data.NightTicks != 10*data.TPS || data.DawnTicks != 5*data.TPS {
		t.Fatalf("the cycle is %d, %d, %d and %d ticks, want 10, 5, 10 and 5 seconds", data.DayTicks, data.DuskTicks, data.NightTicks, data.DawnTicks)
	}
}

func TestNightCutsVisionToAThirdRoundedDown(t *testing.T) {
	for _, c := range []struct {
		radius   int
		darkness float64
		want     float64
	}{
		{36, 0, 36},
		{36, 1, 12},
		{36, 0.5, 24},
		{10, 1, 3},
		{10, 0.5, 6.5},
		{2, 1, 0},
	} {
		if got := ecs.VisionReach(c.radius, c.darkness); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("radius %d at darkness %v reaches %v, want %v", c.radius, c.darkness, got, c.want)
		}
	}
}

func TestVisionSeesLessAtNight(t *testing.T) {
	x := testutil.NewECS(t)
	viewer := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewVisionComponent(36))
	v, _ := x.Visions.Get(viewer)
	far, near := utils.Vec2i{X: 20}, utils.Vec2i{X: 11}

	x.TickIdx = 0
	visionsystem.Tick(x)
	if !v.CanSee(far) || !v.CanSee(near) {
		t.Fatal("by day, the viewer doesn't see 20 tiles away")
	}

	x.TickIdx = data.DayTicks + data.DuskTicks
	visionsystem.Tick(x)
	if v.CanSee(far) || !v.CanSee(near) {
		t.Fatalf("at night, the viewer sees 20 tiles away %v and 11 tiles away %v; want only 11", v.CanSee(far), v.CanSee(near))
	}
	if v.Radius() != 36 {
		t.Fatalf("night changed the vision's radius to %d", v.Radius())
	}
}
