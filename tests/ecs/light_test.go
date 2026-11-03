package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
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

const nightTick = data.DayTicks + data.DuskTicks

func roofedEverywhere(utils.Vec2i) bool { return true }

func TestATorchsLightFadesWithDistanceToNothingAtItsRange(t *testing.T) {
	m := lightAround(t, skyTerrain{roofed: roofedEverywhere, torches: []utils.Vec2i{{}}}, nightTick)

	for tile, want := range map[utils.Vec2i]uint8{
		{}:           150,
		{X: 1}:       135,
		{X: 5}:       75,
		{X: 3, Y: 4}: 75,
		{X: -9}:      15,
		{X: 7, Y: 7}: 1,
		{Y: 10}:      0,
		{X: 6, Y: 8}: 0,
		{X: 11}:      0,
	} {
		if got := m.At(tile); got != want {
			t.Errorf("%v from the torch, the light is %d, want %d", tile, got, want)
		}
	}
}

func TestATorchCastsSharpShadows(t *testing.T) {
	wall := utils.Vec2i{X: 2}
	m := lightAround(t, skyTerrain{roofed: roofedEverywhere, walls: map[utils.Vec2i]bool{wall: true}, torches: []utils.Vec2i{{}}}, nightTick)

	if got := m.At(wall); got != 120 {
		t.Fatalf("the wall facing the torch is lit %d, want 120", got)
	}
	for _, behind := range []utils.Vec2i{{X: 3}, {X: 6}} {
		if got := m.At(behind); got != 0 {
			t.Errorf("behind the wall at %v, the light is %d, want dark rather than light spread around it", behind, got)
		}
	}
	if got := m.At(utils.Vec2i{X: 3, Y: 2}); got == 0 {
		t.Error("beside the wall's shadow, the torch doesn't light")
	}
}

func TestTheBrightestOfSkyAndTorchLightWins(t *testing.T) {
	torch := utils.Vec2i{X: 2}
	m := lightAround(t, skyTerrain{roofFromX: 0, torches: []utils.Vec2i{torch}}, nightTick)

	if got := m.At(utils.Vec2i{X: -7}); got != data.NightSkyLight {
		t.Errorf("under the night sky, 9 tiles from the torch, the light is %d, want the sky's %d over the torch's 15", got, data.NightSkyLight)
	}
	if got := m.At(utils.Vec2i{X: -3}); got != 75 {
		t.Errorf("under the night sky, 5 tiles from the torch, the light is %d, want the torch's 75", got)
	}
	if got := m.At(torch); got != 150 {
		t.Errorf("on the torch under a roof, the light is %d, want 150", got)
	}

	day := lightAround(t, skyTerrain{roofFromX: 0, torches: []utils.Vec2i{torch}}, 0)
	if got := day.At(utils.Vec2i{X: -1}); got != data.DaySkyLight {
		t.Errorf("in daylight next to the torch, the light is %d, want full daylight", got)
	}
}

func TestATorchOutsideTheSquareStillLightsIt(t *testing.T) {
	m := lightAround(t, skyTerrain{roofed: roofedEverywhere, torches: []utils.Vec2i{{X: 35}}}, nightTick)
	if _, radius := m.Area(); radius != 27 {
		t.Fatalf("the light was worked out %d tiles around, want 27", radius)
	}
	if got := m.At(utils.Vec2i{X: 27}); got != 30 {
		t.Fatalf("at the square's edge, 8 tiles from a torch outside it, the light is %d, want 30", got)
	}
}

// A holder whose own tile is the one given, holding a light emitter with a local offset into the next tile over
func holdLight(t *testing.T, x *ecs.ECSContainer, tile utils.Vec2i, linkKind ecs.LinkKindEnum) (holder, light common.EntityId) {
	t.Helper()
	emitter, err := ecs.NewLightEmitterComponent(150, 10)
	if err != nil {
		t.Fatal(err)
	}
	holder = testutil.AddMovable(t, x, utils.TilePosToWorldPos(tile), false)
	light = testutil.AddMovable(t, x, utils.Vec2f{X: data.TileSize}, false, emitter)
	testutil.Attach(t, x, light, holder, linkKind)
	return holder, light
}

func lightOf(t *testing.T, x *ecs.ECSContainer) *ecs.LightMap {
	t.Helper()
	x.SetTerrain(skyTerrain{Terrain: x.Terrain(), roofed: roofedEverywhere})
	x.TickIdx = nightTick
	m := &ecs.LightMap{}
	x.LightAround(utils.Vec2i{}, 20, m)
	return m
}

func TestAHeldLightShinesFromItsHoldersTileLikeATorch(t *testing.T) {
	x := testutil.NewECS(t)
	holdLight(t, x, utils.Vec2i{X: 2}, ecs.Link_Held)
	m := lightOf(t, x)

	for tile, want := range map[utils.Vec2i]uint8{{X: 2}: 150, {X: 7}: 75, {X: -3}: 75, {X: 12}: 0} {
		if got := m.At(tile); got != want {
			t.Errorf("at %v, held light 150 from (2, 0) lights %d, want %d", tile, got, want)
		}
	}
}

func TestOnlyAnEnabledHeldLightShines(t *testing.T) {
	cases := []struct {
		name  string
		setup func(x *ecs.ECSContainer)
	}{
		{"put away", func(x *ecs.ECSContainer) {
			_, light := holdLight(t, x, utils.Vec2i{}, ecs.Link_Held)
			x.SetDisabled(light, true)
		}},
		{"attached rather than held", func(x *ecs.ECSContainer) {
			holdLight(t, x, utils.Vec2i{}, ecs.Link_Attached)
		}},
		{"lying loose", func(x *ecs.ECSContainer) {
			emitter, err := ecs.NewLightEmitterComponent(150, 10)
			if err != nil {
				t.Fatal(err)
			}
			testutil.AddMovable(t, x, utils.Vec2f{}, false, emitter)
		}},
	}
	for _, c := range cases {
		x := testutil.NewECS(t)
		c.setup(x)
		if got := lightOf(t, x).At(utils.Vec2i{}); got != 0 {
			t.Errorf("a light %s lights %d", c.name, got)
		}
	}
}

func TestLightEmittersNeedLight(t *testing.T) {
	for _, c := range []struct {
		level uint8
		reach int
	}{{0, 10}, {150, 0}, {150, -1}} {
		if _, err := ecs.NewLightEmitterComponent(c.level, c.reach); err == nil {
			t.Errorf("light %d up to %d tiles away was accepted", c.level, c.reach)
		}
	}
}
