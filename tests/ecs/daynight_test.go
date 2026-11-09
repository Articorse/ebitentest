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

func TestTheCycleIsTwentyFourMinutesOfTenDayTwoDuskTenNightTwoDawn(t *testing.T) {
	if data.DayTicks != 10*60*data.TPS || data.DuskTicks != 2*60*data.TPS || data.NightTicks != 10*60*data.TPS || data.DawnTicks != 2*60*data.TPS {
		t.Fatalf("the cycle is %d, %d, %d and %d ticks, want 10, 2, 10 and 2 minutes", data.DayTicks, data.DuskTicks, data.NightTicks, data.DawnTicks)
	}
}

func TestTheSkyLightsByDayAndDimsAtNight(t *testing.T) {
	for _, c := range []struct {
		darkness float64
		want     uint8
	}{
		{0, data.DaySkyLight},
		{1, data.NightSkyLight},
		{0.5, uint8((data.DaySkyLight + data.NightSkyLight + 1) / 2)},
	} {
		if got := ecs.SkyLight(c.darkness); got != c.want {
			t.Errorf("at darkness %v the sky gives light %d, want %d", c.darkness, got, c.want)
		}
	}
}

// Open to the sky everywhere but where it's roofed, which by default is from roofFromX on, with walls and torches
// giving 150 light up to 10 tiles away where told
type skyTerrain struct {
	ecs.Terrain
	roofFromX int
	roofed    func(utils.Vec2i) bool
	walls     map[utils.Vec2i]bool
	torches   []utils.Vec2i
}

func (s skyTerrain) IsOpenToSky(t utils.Vec2i) bool {
	if s.roofed != nil {
		return !s.roofed(t)
	}
	return t.X < s.roofFromX
}
func (s skyTerrain) IsOpaqueTile(t utils.Vec2i) bool { return s.walls[t] }
func (s skyTerrain) LightSources(_, _ utils.Vec2i, lit func(utils.Vec2i, uint8, int)) {
	for _, t := range s.torches {
		lit(t, 150, 10)
	}
}

func TestVisionSeesLessAtNightAndNothingDeepUnderARoof(t *testing.T) {
	x := testutil.NewECS(t)
	x.SetTerrain(skyTerrain{Terrain: x.Terrain(), roofFromX: 30})
	viewer := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewVisionComponent(data.PlayerVisionRangeTiles, 1))
	v, _ := x.Visions.Get(viewer)
	far, near := utils.Vec2i{X: 20}, utils.Vec2i{X: 11}
	underTheEdge, deepUnder := utils.Vec2i{X: 31}, utils.Vec2i{X: 40}

	x.TickIdx = 0
	visionsystem.Tick(x)
	if !v.CanSee(far) || !v.CanSee(near) {
		t.Fatal("by day, the viewer doesn't see 20 tiles away")
	}
	if !v.CanSee(underTheEdge) || v.CanSee(deepUnder) {
		t.Fatalf("by day, the viewer sees just under the roof's edge %v and deep under it %v; want only the edge",
			v.CanSee(underTheEdge), v.CanSee(deepUnder))
	}

	x.TickIdx = data.DayTicks + data.DuskTicks
	visionsystem.Tick(x)
	if v.CanSee(far) || !v.CanSee(near) {
		t.Fatalf("at night, the viewer sees 20 tiles away %v and 11 tiles away %v; want only 11", v.CanSee(far), v.CanSee(near))
	}
}
