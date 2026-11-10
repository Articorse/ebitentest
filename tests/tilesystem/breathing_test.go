package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"strings"
	"testing"
)

// Something of the body that does nothing but breathe
func breather(t *testing.T, s *testutil.Session, tile utils.Vec2i, kind data.BodyEnum) common.EntityId {
	t.Helper()
	return testutil.Add(t, s.ECS,
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(utils.TilePosToWorldPos(tile), 1, 0),
		ecs.NewBodyComponent(kind),
	)
}

func TestBreathingTurnsARoomsOxygenIntoCarbonDioxideAtItsBodysRate(t *testing.T) {
	for _, c := range []struct {
		name string
		kind data.BodyEnum
		// A step's breath, in milligrams
		step int64
	}{
		{"a humanoid", data.Body_Humanoid, 13889 * data.AirStepTicks / data.TPS},
		{"a giant", data.Body_Giant, 55556 * data.AirStepTicks / data.TPS},
	} {
		t.Run(c.name, func(t *testing.T) {
			testutil.TempWorld(t)
			s := testutil.StartSession(t)
			middle := sealedRoom(t, s, 3)
			before := airAt(t, s, middle)
			breather(t, s, middle, c.kind)
			s.Tick(60 * data.TPS)
			breathed := c.step * 60 * data.TPS / data.AirStepTicks / 25
			after := airAt(t, s, middle)
			if got := before[data.Gas_Oxygen] - after[data.Gas_Oxygen]; abs(int(got-breathed)) > 1 {
				t.Errorf("a minute's breathing took %d mg of oxygen per tile, want %d", got, breathed)
			}
			if got := after[data.Gas_CarbonDioxide] - before[data.Gas_CarbonDioxide]; abs(int(got-breathed)) > 1 {
				t.Errorf("a minute's breathing gave %d mg of carbon dioxide per tile, want %d", got, breathed)
			}
		})
	}
}

func TestBreathingOutdoorsChangesNothingAndNeedsNothingDone(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	outside := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 1, anywhere)
	e := breather(t, s, outside, data.Body_Humanoid)
	s.Tick(10 * data.TPS)
	if got, want := airAt(t, s, outside), planet(t, data.Planet_Earthlike).Atmosphere; got != want {
		t.Fatalf("outdoors after breathing, the air is %v, want %v", got, want)
	}
	if s.ECS.InBadAir(e) {
		t.Fatal("Earthlike's outdoor air is bad to breathe")
	}
}

func TestAVacuumIsBadToBreatheButNotForWhatDoesntBreathe(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	s.Chunks.SetPlanet(planet(t, data.Planet_Moon))
	outside := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 1, anywhere)
	e := breather(t, s, outside, data.Body_Humanoid)
	if !s.ECS.InBadAir(e) {
		t.Fatal("a vacuum isn't bad to breathe")
	}
	if status, err := drawsystem.BodyStatusText(e, s.ECS); err != nil || !strings.Contains(status, "Bad air") {
		t.Fatalf("the body status is %q (%v), without bad air", status, err)
	}
	stuffy := constant(t, 15, 1)
	stuffy.Atmosphere[data.Gas_CarbonDioxide] = data.SafeCarbonDioxideMg + data.MilligramsPerGram
	s.Chunks.SetPlanet(stuffy)
	if !s.ECS.InBadAir(e) {
		t.Fatal("air with too much carbon dioxide isn't bad to breathe")
	}
	stone := testutil.Add(t, s.ECS, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.TilePosToWorldPos(outside), 1, 0))
	if s.ECS.InBadAir(stone) {
		t.Fatal("something with no body is in bad air")
	}
}

func TestOnADoorItsTheAirOfTheSideItsOnThatsBreathed(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := sealedRoom(t, s, 3)
	gap := at(middle, 3, 0)
	if destroyed, _ := s.Chunks.DamageTile(gap, 1000); !destroyed {
		t.Fatal("the wall couldn't be knocked down")
	}
	buildDoor(t, s, gap, data.Faction_Colony)
	center := utils.TilePosToWorldPos(gap)
	if got := s.ECS.AirTile(center.Add(utils.Vec2f{X: -3})); got != at(gap, -1, 0) {
		t.Fatalf("a little inside the door, it breathes the air of %v, want the room's", got)
	}
	if got := s.ECS.AirTile(center.Add(utils.Vec2f{X: 3, Y: 1})); got != at(gap, 1, 0) {
		t.Fatalf("a little outside the door, it breathes the air of %v, want the outdoors'", got)
	}
}

// A room the colonist is shut in with no air, with a door of the colony's onto ground that's outdoors
func shutInWithNoAir(t *testing.T) (s *testutil.Session, colonist common.EntityId, middle, gap utils.Vec2i) {
	t.Helper()
	s, _, colonist = colonistAtWork(t)
	middle = doorYard(t, s, colonist)
	gap = at(middle, 0, 3)
	s.Chunks.SetPlanet(planet(t, data.Planet_Moon))
	wall(t, s, ring(middle, 3, gap)...)
	buildDoor(t, s, gap, data.Faction_Colony)
	roof(t, s, rect(at(middle, -2, -2), at(middle, 2, 2)))
	roomAt(t, s, middle)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(middle))
	return s, colonist, middle, gap
}

func TestAColonistInBadAirGetsToTheClosestSafeAir(t *testing.T) {
	s, colonist, middle, _ := shutInWithNoAir(t)
	s.Chunks.SetPlanet(planet(t, data.Planet_Earthlike))
	if !live(t, s, data.TPS, func() bool { return colonistJob(s, colonist).Emergency.Kind == inputsources.Emergency_GetToAir }) {
		t.Fatal("the colonist in a room with no air wasn't told to get to air")
	}
	if !live(t, s, 30*data.TPS, func() bool {
		air, ok := s.ECS.AirOf(colonist)
		return ok && ecs.SafeAir(air) && colonistJob(s, colonist).Emergency.Kind == inputsources.Emergency_None
	}) {
		t.Fatalf("the colonist didn't get to safe air, it's on %v, with the middle of the room on %v", colonistTile(t, s, colonist), middle)
	}
}

func TestAColonistGetsToSafeAirBeforeAirItCanBreatheAndToThatWithNoSafeAir(t *testing.T) {
	for _, c := range []struct {
		name string
		safe bool
	}{{"with safe air beyond", true}, {"with no safe air", false}} {
		t.Run(c.name, func(t *testing.T) { getsToAir(t, c.safe) })
	}
}

// Shut in with no air, with a door onto a room of thin air beyond, which has a door onto safe air beyond it if safe
func getsToAir(t *testing.T, safe bool) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	thin := constant(t, 15, 1)
	thin.Atmosphere[data.Gas_Oxygen] = 200 * data.MilligramsPerGram
	thin.Atmosphere[data.Gas_CarbonDioxide] = 0
	s.Chunks.SetPlanet(thin)
	lower := []utils.Vec2i{}
	for _, tile := range ring(middle, 3) {
		if tile.Y > middle.Y {
			lower = append(lower, tile)
		}
	}
	if safe {
		lower = slices.DeleteFunc(lower, func(tile utils.Vec2i) bool { return tile == at(middle, 0, 3) })
	}
	wall(t, s, append(lower, at(middle, -3, 0), at(middle, -2, 0), at(middle, -1, 0), at(middle, 1, 0), at(middle, 2, 0), at(middle, 3, 0))...)
	buildDoor(t, s, middle, data.Faction_Colony)
	if safe {
		buildDoor(t, s, at(middle, 0, 3), data.Faction_Colony)
	}
	roof(t, s, rect(at(middle, -2, 1), at(middle, 2, 2)))
	roomAt(t, s, at(middle, 0, 1))

	s.Chunks.SetPlanet(planet(t, data.Planet_Moon))
	upper := []utils.Vec2i{}
	for _, tile := range ring(middle, 3) {
		if tile.Y < middle.Y {
			upper = append(upper, tile)
		}
	}
	wall(t, s, upper...)
	roof(t, s, rect(at(middle, -2, -2), at(middle, 2, -1)))
	roomAt(t, s, at(middle, 0, -1))
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(middle, 0, -2)))
	if safe {
		s.Chunks.SetPlanet(planet(t, data.Planet_Earthlike))
	}

	if !live(t, s, data.TPS, func() bool { return colonistJob(s, colonist).Emergency.Kind == inputsources.Emergency_GetToAir }) {
		t.Fatal("the colonist in a room with no air wasn't told to get to air")
	}
	beyond := middle.Y
	if safe {
		beyond = middle.Y + 3
	}
	if got := colonistJob(s, colonist).Emergency.Tile; got.Y <= beyond {
		t.Fatalf("the colonist was told to get to %v, want the air beyond y %d", got, beyond)
	}
	if !live(t, s, 20*data.TPS, func() bool {
		return colonistTile(t, s, colonist).Y > beyond && colonistJob(s, colonist).Emergency.Kind == inputsources.Emergency_None
	}) {
		t.Fatalf("the colonist didn't get through to the air beyond y %d, it's on %v", beyond, colonistTile(t, s, colonist))
	}
}

func TestAColonistWithNoAirItCanBreatheToGetToStaysPut(t *testing.T) {
	s, colonist, _, _ := shutInWithNoAir(t)
	live(t, s, 2*data.TPS, func() bool {
		if colonistJob(s, colonist).Emergency.Kind == inputsources.Emergency_GetToAir {
			t.Fatal("the colonist was told to get to air with none to be had")
		}
		return false
	})
}
