package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func lightAround(t *testing.T, terrain skyTerrain, tick uint64) *ecs.LightMap {
	t.Helper()
	x := testutil.NewECS(t)
	terrain.Terrain = x.Terrain()
	x.SetTerrain(terrain)
	x.TickIdx = tick
	m := &ecs.LightMap{}
	x.LightAround(utils.Vec2i{}, 20, m)
	return m
}

func TestSkyLightSpreadsUnderRoofsLosingSomeWithEachTile(t *testing.T) {
	m := lightAround(t, skyTerrain{roofFromX: 0}, 0)
	for steps := range 10 {
		want := max(data.DaySkyLight-steps*data.LightFalloffPerTile, 0)
		if got := m.At(utils.Vec2i{X: steps - 1, Y: 3}); int(got) != want {
			t.Errorf("%d tiles from the sky, the light is %d, want %d", steps, got, want)
		}
	}
}

func TestLightSpreadsDiagonally(t *testing.T) {
	walls := map[utils.Vec2i]bool{}
	// A wall along x = 1, but for the tile at y = 0, which the light has to squeeze through diagonally
	for y := -20; y <= 20; y++ {
		if y != 0 {
			walls[utils.Vec2i{X: 1, Y: y}] = true
		}
	}
	walls[utils.Vec2i{X: 1, Y: 0}] = true
	walls[utils.Vec2i{X: 2, Y: 0}] = true
	delete(walls, utils.Vec2i{X: 1, Y: 1})
	m := lightAround(t, skyTerrain{roofFromX: 1, walls: walls}, 0)

	// Through the gap at (1, 1), then diagonally past the wall at (2, 0)
	if got, want := m.At(utils.Vec2i{X: 3, Y: 0}), uint8(data.DaySkyLight-3*data.LightFalloffPerTile); got != want {
		t.Fatalf("past a diagonal gap, the light is %d, want %d", got, want)
	}
}

func TestWallsAreLitButDontPassLightOn(t *testing.T) {
	walls := map[utils.Vec2i]bool{}
	for y := -20; y <= 20; y++ {
		walls[utils.Vec2i{X: 0, Y: y}] = true
	}
	m := lightAround(t, skyTerrain{roofFromX: 0, walls: walls}, 0)

	if got, want := m.At(utils.Vec2i{X: 0, Y: 5}), uint8(data.DaySkyLight-data.LightFalloffPerTile); got != want {
		t.Fatalf("the wall facing the sky is lit %d, want %d", got, want)
	}
	if got := m.At(utils.Vec2i{X: 1, Y: 5}); got != 0 {
		t.Fatalf("behind the wall, the light is %d, want dark", got)
	}
}

func TestAWallOpenToTheSkyDoesntLightTheRoomBehindIt(t *testing.T) {
	walls := map[utils.Vec2i]bool{}
	for y := -20; y <= 20; y++ {
		walls[utils.Vec2i{X: -1, Y: y}] = true
	}
	m := lightAround(t, skyTerrain{roofFromX: 0, walls: walls}, 0)

	if got := m.At(utils.Vec2i{X: -1, Y: 2}); got != data.DaySkyLight {
		t.Fatalf("the wall under the open sky is lit %d, want full daylight", got)
	}
	if got := m.At(utils.Vec2i{X: 0, Y: 2}); got != 0 {
		t.Fatalf("the roofed room behind the wall is lit %d, want dark", got)
	}
}

func TestNightSkyLightOnlyReachesTheFirstRoofedTile(t *testing.T) {
	m := lightAround(t, skyTerrain{roofFromX: 0}, data.DayTicks+data.DuskTicks)
	if got, want := m.At(utils.Vec2i{X: 0}), uint8(data.NightSkyLight-data.LightFalloffPerTile); got != want {
		t.Fatalf("the first roofed tile at night is lit %d, want %d", got, want)
	}
	if got := m.At(utils.Vec2i{X: 1}); got != 0 {
		t.Fatalf("the second roofed tile at night is lit %d, want dark", got)
	}
}

// Light can come from sky beyond the square asked for, which has to be looked at too
func TestLightComesInFromBeyondTheSquareAsked(t *testing.T) {
	m := lightAround(t, skyTerrain{roofed: func(t utils.Vec2i) bool { return t.X < 23 }}, 0)
	if got, want := m.At(utils.Vec2i{X: 20}), uint8(data.DaySkyLight-3*data.LightFalloffPerTile); got != want {
		t.Fatalf("3 tiles from sky just past the square, the light is %d, want %d", got, want)
	}
}

func TestALightMapNotYetWorkedOutIsDark(t *testing.T) {
	m := &ecs.LightMap{}
	if got := m.At(utils.Vec2i{}); got != 0 {
		t.Fatalf("an empty light map gives light %d, want dark", got)
	}
}
