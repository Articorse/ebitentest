package tilesystem_test

import (
	"bytes"
	"ebittest/climate"
	"ebittest/data"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"encoding/gob"
	"image/color"
	"os"
	"slices"
	"testing"
)

func planet(t *testing.T, id data.PlanetEnum) climate.Planet {
	t.Helper()
	p, err := climate.PlanetOf(testutil.Planets(t), id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func airAt(t *testing.T, s *testutil.Session, tile utils.Vec2i) climate.Air {
	t.Helper()
	air, ok := s.Chunks.AirAt(tile)
	if !ok {
		t.Fatalf("%v has no air", tile)
	}
	return air
}

// Walled round, r tiles from the middle, and roofed over all but the gaps, near the player
func sealedRoom(t *testing.T, s *testutil.Session, r int, gaps ...utils.Vec2i) utils.Vec2i {
	t.Helper()
	middle := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), r, anywhere)
	wall(t, s, ring(middle, r)...)
	roof(t, s, rect(at(middle, 1-r, 1-r), at(middle, r-1, r-1)), gaps...)
	// Worked out now, with the outdoors' air as it is now, rather than when it's first asked about
	roomAt(t, s, middle)
	return middle
}

func unroof(t *testing.T, s *testutil.Session, tile utils.Vec2i) {
	t.Helper()
	s.Chunks.SetArea(data.Area_Roof, tile, tile, false)
	s.Chunks.SetArea(data.Area_NoRoof, tile, tile, true)
	if _, done := s.Chunks.WorkOnRoof(tile, data.RoofWork); !done {
		t.Fatalf("couldn't take the roof over %v down", tile)
	}
}

// Over the tiles but the gaps, after the walls and doors beside them, which hold the roof up, building outward from
// them as far as it's held up
func roof(t *testing.T, s *testutil.Session, tiles []utils.Vec2i, gaps ...utils.Vec2i) {
	t.Helper()
	todo := slices.DeleteFunc(slices.Clone(tiles), func(tile utils.Vec2i) bool { return slices.Contains(gaps, tile) })
	supports := []utils.Vec2i{}
	for _, tile := range todo {
		for _, n := range []utils.Vec2i{at(tile, -1, 0), at(tile, 1, 0), at(tile, 0, -1), at(tile, 0, 1)} {
			layers, _ := s.Chunks.TileAt(n)
			_, isDoor := s.Chunks.DoorAt(n)
			if (layers[data.TileLayer_Wall] != data.Tile_None || isDoor) && roofOver(s, n) == data.Roof_None && !slices.Contains(todo, n) && !slices.Contains(supports, n) {
				supports = append(supports, n)
			}
		}
	}
	todo = append(supports, todo...)
	for len(todo) > 0 {
		left := todo[:0]
		for _, tile := range todo {
			s.Chunks.SetArea(data.Area_Roof, tile, tile, true)
			if _, done := s.Chunks.WorkOnRoof(tile, data.RoofWork); !done {
				left = append(left, tile)
			}
		}
		if len(left) == len(todo) {
			t.Fatalf("couldn't roof %v, which nothing holds up", left)
		}
		todo = left
	}
}

func TestANewRoomHasTheOutdoorsAir(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := sealedRoom(t, s, 3)
	if got, want := airAt(t, s, middle), planet(t, data.Planet_Earthlike).Atmosphere; got != want {
		t.Fatalf("the new room's air is %v per tile, want the outdoors' %v", got, want)
	}
	if got, want := s.Chunks.AirTemperature(middle), s.Chunks.AmbientTemperature(middle, s.ECS.TickIdx); abs(int(got-want)) > 100 {
		t.Fatalf("the new room is %v, want about the %v it is outdoors", got.Celsius(), want.Celsius())
	}
	if got, want := airAt(t, s, at(middle, 5, 0)), planet(t, data.Planet_Earthlike).Atmosphere; got != want {
		t.Fatalf("the air outdoors is %v per tile, want the planet's %v", got, want)
	}
	if want := []string{"  Oxygen 900 g (90%)", "  Carbon dioxide 100 g (10%)"}; !slices.Contains(s.Chunks.AirDebugLines(middle), want[0]) || !slices.Contains(s.Chunks.AirDebugLines(middle), want[1]) {
		t.Fatalf("the tooltip shows %v, want it to show %v", s.Chunks.AirDebugLines(middle), want)
	}
}

func TestARoomBuiltInAnotherHasItsAirAndOneBuiltOutdoorsTheOutdoors(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := sealedRoom(t, s, 3)
	earth := airAt(t, s, middle)
	s.Chunks.SetPlanet(planet(t, data.Planet_Moon))

	wall(t, s, ring(middle, 1)...)
	if got := airAt(t, s, middle); got != earth {
		t.Fatalf("the room built in another has %v per tile, want the %v of the one it's in", got, earth)
	}
	if got := airAt(t, s, at(middle, 0, 2)); got != earth {
		t.Fatalf("what's left of the room it's built in has %v per tile, want the %v it had", got, earth)
	}

	outdoors := ringSpot(t, s, at(middle, 10, 0), 2, func(m utils.Vec2i) bool { return abs(m.X-middle.X) > 6 || abs(m.Y-middle.Y) > 6 })
	wall(t, s, ring(outdoors, 2)...)
	if got := airAt(t, s, outdoors); got.Total() != 0 {
		t.Fatalf("a room built outdoors on the moon has %v per tile, want none", got)
	}
	if lines := s.Chunks.AirDebugLines(outdoors); !slices.Contains(lines, "Vacuum") {
		t.Fatalf("the tooltip shows %v, want vacuum", lines)
	}
}

func TestASealedRoomKeepsItsAirAndAnUnroofedTileLetsItOutInAboutThirtySeconds(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := sealedRoom(t, s, 3)
	sealed := airAt(t, s, middle)[data.Gas_Oxygen]
	s.Chunks.SetPlanet(planet(t, data.Planet_Moon))
	s.Tick(10 * data.TPS)
	if got := airAt(t, s, middle)[data.Gas_Oxygen]; got != sealed {
		t.Fatalf("the sealed room's oxygen went from %d to %d mg per tile", sealed, got)
	}

	unroof(t, s, at(middle, 2, 2))
	s.Tick(5 * data.TPS)
	if got := airAt(t, s, middle)[data.Gas_Oxygen]; got < sealed/2 || got > sealed*3/4 {
		t.Fatalf("5 seconds after a tile was opened to the sky, the room has %d of %d mg of oxygen per tile, want about 60%%", got, sealed)
	}
	s.Tick(25 * data.TPS)
	if got := airAt(t, s, middle)[data.Gas_Oxygen]; got > sealed/10 {
		t.Fatalf("30 seconds after a tile was opened to the sky, the room still has %d of %d mg of oxygen per tile", got, sealed)
	}
}

func TestAnUnroofedRoomIsNeverEmptiedPastTheOutdoors(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 1, anywhere)
	wall(t, s, ring(middle, 1)...)
	before := airAt(t, s, middle)[data.Gas_Oxygen]
	s.Chunks.SetPlanet(planet(t, data.Planet_Moon))
	s.Tick(data.AirStepTicks)
	if got := airAt(t, s, middle)[data.Gas_Oxygen]; got < before/2-1 || got >= before {
		t.Fatalf("a step after the outdoors became a vacuum, the one tile room open to the sky has %d of %d mg of oxygen, want half", got, before)
	}
}

func TestAirFlowsThroughAnOpenDoorAndNotAClosedOne(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	gap := at(middle, 0, 3)
	wall(t, s, ring(middle, 3, gap)...)
	buildDoor(t, s, gap, data.Faction_Colony)
	roof(t, s, rect(at(middle, -2, -2), at(middle, 2, 2)))
	s.ECS.SetDisabled(colonist, true)
	sealed := airAt(t, s, middle)[data.Gas_Oxygen]
	s.Chunks.SetPlanet(planet(t, data.Planet_Moon))

	live(t, s, 5*data.TPS, func() bool { return false })
	if got := airAt(t, s, middle)[data.Gas_Oxygen]; got != sealed {
		t.Fatalf("with the door closed, the room's oxygen went from %d to %d mg per tile", sealed, got)
	}
	s.ECS.SetDisabled(colonist, false)
	live(t, s, 10*data.TPS, func() bool {
		testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(gap, 0, 1)))
		return false
	})
	if !door(t, s, gap).Open {
		t.Fatal("the door didn't open for the colonist by it")
	}
	if got := airAt(t, s, middle)[data.Gas_Oxygen]; got > sealed/2 {
		t.Fatalf("10 seconds after the door opened onto a vacuum, the room has %d of %d mg of oxygen per tile", got, sealed)
	}
}

func TestAirComingInMixesItsWarmthIn(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := sealedRoom(t, s, 3)
	hot := planet(t, data.Planet_Earthlike)
	hot.SpawnCelsius, hot.NorthCelsius, hot.SouthCelsius = 100, 100, 100
	hot.NoiseSwing, hot.SeasonSwing, hot.DayNightSwing = 0, 0, 0
	s.Chunks.SetPlanet(hot)
	before := s.Chunks.AirTemperature(middle)
	s.Tick(10 * data.TPS)
	if s.Chunks.AirTemperature(middle) != before {
		t.Fatal("the sealed room warmed up with no way in for the air outdoors")
	}

	hot.Atmosphere[data.Gas_Oxygen] += hot.Atmosphere.Total()
	s.Chunks.SetPlanet(hot)
	unroof(t, s, at(middle, 2, 2))
	s.Tick(60 * data.TPS)
	// Half its air is what it had, and half came in at 100 °C
	if got, want := s.Chunks.AirTemperature(middle).Celsius(), (before.Celsius()+100)/2; got < want-5 || got > want+5 {
		t.Fatalf("with as much air again let in at 100 °C, the room is %v °C, want about %v", got, want)
	}
}

func TestARoomsAirIsSaved(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := sealedRoom(t, s, 3)
	s.Chunks.SetPlanet(planet(t, data.Planet_Moon))
	wall(t, s, ring(middle, 1)...)
	air, temperature := airAt(t, s, middle), s.Chunks.AirTemperature(middle)
	s.Commit()

	r := testutil.StartSession(t)
	if got := airAt(t, r, middle); got != air || r.Chunks.AirTemperature(middle) != temperature {
		t.Fatalf("after a restart the room's air is %v at %v, want %v at %v", got, r.Chunks.AirTemperature(middle), air, temperature)
	}
}

func TestTheOxygenOverlayRunsFromRedThroughYellowToGreen(t *testing.T) {
	a := uint8(data.OverlayOxygenAlpha)
	for _, c := range []struct {
		name  string
		grams int64
		want  color.RGBA
	}{
		{"with none", 0, color.RGBA{R: a, A: a}},
		{"at the red end", 100, color.RGBA{R: a, A: a}},
		{"a quarter of the way", 200, color.RGBA{R: a, G: a / 2, A: a}},
		{"halfway", 300, color.RGBA{R: a, G: a, A: a}},
		{"at the green end", 500, color.RGBA{G: a, A: a}},
		{"with plenty", 900, color.RGBA{G: a, A: a}},
	} {
		if got := drawsystem.OxygenColor(c.grams * data.MilligramsPerGram); got != c.want {
			t.Errorf("%s it's %v, want %v", c.name, got, c.want)
		}
	}
}

// The rooms file as gob sees it, so that it can be written as it was before rooms had air
type roomsFileBeforeAir struct {
	NextId uint64
	Chunks []struct {
		X, Y   int
		Pieces []struct {
			Anchor int32
			Tiles  int
			Room   uint64
		}
		Edges []int32
	}
}

func TestARoomSavedBeforeRoomsHadAirHasTheOutdoorsAir(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	s.Chunks.SetPlanet(planet(t, data.Planet_Moon))
	middle := sealedRoom(t, s, 3)
	s.Chunks.SetPlanet(planet(t, data.Planet_Earthlike))
	s.Commit()

	file, err := os.ReadFile(data.RoomsFilePath)
	if err != nil {
		t.Fatal(err)
	}
	var old roomsFileBeforeAir
	if err := gob.NewDecoder(bytes.NewReader(file)).Decode(&old); err != nil {
		t.Fatal(err)
	}
	var rewritten bytes.Buffer
	if err := gob.NewEncoder(&rewritten).Encode(old); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data.RoomsFilePath, rewritten.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	r := testutil.StartSession(t)
	if got, want := airAt(t, r, middle), planet(t, data.Planet_Earthlike).Atmosphere; got != want {
		t.Fatalf("the room saved without air has %v per tile, want the outdoors' %v", got, want)
	}
}

func TestARoomThatChangesKeepsItsAirAndWarmth(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := sealedRoom(t, s, 3)
	air, warmth := airAt(t, s, middle), s.Chunks.AirTemperature(middle)
	s.Chunks.SetPlanet(constant(t, 100, 1))
	wall(t, s, at(middle, -2, 1), at(middle, -1, 1), at(middle, 0, 1), at(middle, 1, 1), at(middle, 2, 1))
	if got := airAt(t, s, middle); got != air || s.Chunks.AirTemperature(middle) != warmth {
		t.Fatalf("split, the room has %v at %v, want the %v at %v it had", got, s.Chunks.AirTemperature(middle), air, warmth)
	}
}

// Earthlike's air, as many times over, at the same temperature everywhere and always
func constant(t *testing.T, celsius float64, times int64) climate.Planet {
	t.Helper()
	p := planet(t, data.Planet_Earthlike)
	p.SpawnCelsius, p.NorthCelsius, p.SouthCelsius = celsius, celsius, celsius
	p.NoiseSwing, p.SeasonSwing, p.DayNightSwing = 0, 0, 0
	for g := range p.Atmosphere {
		p.Atmosphere[g] *= times
	}
	return p
}

func TestAirFromAnotherRoomComesInAsWarmAsThatRoom(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	s.ECS.SetDisabled(colonist, true)
	middle := doorYard(t, s, colonist)
	wall(t, s, ring(middle, 3)...)
	wall(t, s, at(middle, -2, 0), at(middle, -1, 0), at(middle, 1, 0), at(middle, 2, 0))
	buildDoor(t, s, middle, data.Faction_Colony)
	above, below := at(middle, 0, -1), at(middle, 0, 1)
	roof(t, s, rect(at(middle, -2, -2), at(middle, 2, -1)))
	roomAt(t, s, above)
	s.Chunks.SetPlanet(constant(t, 100, 2))
	live(t, s, 60*data.TPS, func() bool { return false })
	if door(t, s, middle).Open {
		t.Fatal("the door opened with no one by it")
	}
	roof(t, s, rect(at(middle, -2, 1), at(middle, 2, 2)))
	s.Chunks.SetPlanet(constant(t, -50, 2))
	// As much air again as it had came in at 100 °C
	if got := s.Chunks.AirTemperature(below).Celsius(); got < 45 {
		t.Fatalf("the room left open to the hot outdoors is %v °C", got)
	}
	before := s.Chunks.AirTemperature(above).Celsius()
	if before > 30 {
		t.Fatalf("the room roofed over is %v °C, warmed with no way in", before)
	}

	s.ECS.SetDisabled(colonist, false)
	live(t, s, 30*data.TPS, func() bool {
		testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(middle, 1, 1)))
		return false
	})
	// A third of its air came in from the hot room, rather than from the cold outdoors
	if got := s.Chunks.AirTemperature(above).Celsius(); got < before+8 {
		t.Fatalf("with the door open onto the hot room, the room went from %v to %v °C, want it warmed by the air that came in", before, got)
	}
}
