package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

// How much of the kind lies on the tiles
func lying(s *testutil.Session, tiles []utils.Vec2i, kind data.ItemEnum) int {
	n := 0
	for _, t := range tiles {
		if on, ok := s.Chunks.GroundStack(t); ok && on.Kind == kind {
			n += on.Amount
		}
	}
	return n
}

func TestAGasDropsOutOfAColdRoomsAirAsItsItemOntoTheRoomsFloor(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	// Colder than carbon dioxide condenses at, but not oxygen
	s.Chunks.SetPlanet(constant(t, -100, 1))
	middle := sealedRoom(t, s, 3)
	inside, around := rect(at(middle, -2, -2), at(middle, 2, 2)), ring(middle, 4)
	before := airAt(t, s, middle)

	s.Tick(30 * data.TPS)
	after := airAt(t, s, middle)
	if after[data.Gas_Oxygen] != before[data.Gas_Oxygen] {
		t.Fatalf("the oxygen went from %d to %d mg per tile, above where it condenses", before[data.Gas_Oxygen], after[data.Gas_Oxygen])
	}
	if after[data.Gas_CarbonDioxide] > before[data.Gas_CarbonDioxide]/4 {
		t.Fatalf("after 30 seconds, the carbon dioxide only went from %d to %d mg per tile", before[data.Gas_CarbonDioxide], after[data.Gas_CarbonDioxide])
	}
	// All 25 tiles' worth, to within what's left over from the air being given per tile
	condensed := int(before[data.Gas_CarbonDioxide]-after[data.Gas_CarbonDioxide]) * 25
	if got := lying(s, inside, data.Item_CarbonDioxide); got < condensed-25 || got > condensed+25 {
		t.Fatalf("%d mg of carbon dioxide lie in the room, want the %d mg that dropped out of its air", got, condensed)
	}
	if lying(s, around, data.Item_CarbonDioxide) != 0 {
		t.Fatal("carbon dioxide dropped out of the room's air outside it")
	}
	s.Tick(30 * data.TPS)
	if got, want := lying(s, inside, data.Item_CarbonDioxide), int(before[data.Gas_CarbonDioxide])*25; got != want {
		t.Fatalf("after a minute, %d mg of carbon dioxide lie in the room, want every last milligram of its %d mg", got, want)
	}
}

func TestAGasesItemGoesBackIntoTheAirOfARoomWarmerThanItCondensesAt(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := sealedRoom(t, s, 3)
	before := airAt(t, s, middle)
	s.Chunks.SetGroundStack(middle, ecs.GroundStack{Kind: data.Item_CarbonDioxide, Amount: 25 * data.MilligramsPerKg})

	s.Tick(data.AirStepTicks)
	if on, _ := s.Chunks.GroundStack(middle); on.Amount != 25*kg*15/16 {
		t.Fatalf("a step later, %d mg of the 25 kg are left, want fifteen sixteenths", on.Amount)
	}
	s.Tick(30 * data.TPS)
	if _, ok := s.Chunks.GroundStack(middle); ok {
		t.Fatal("the carbon dioxide hasn't all gone back into the air after 30 seconds")
	}
	if got, want := airAt(t, s, middle)[data.Gas_CarbonDioxide]-before[data.Gas_CarbonDioxide], int64(data.MilligramsPerKg); got != want {
		t.Fatalf("the room's air has %d mg more carbon dioxide per tile, want the %d that was lying in it", got, want)
	}
}

func TestAGasesItemOutdoorsGoesIntoThePlanetsAir(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tile := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 1, anywhere)
	s.Chunks.SetGroundStack(tile, ecs.GroundStack{Kind: data.Item_Oxygen, Amount: 3 * data.MilligramsPerKg})
	s.Tick(20 * data.TPS)
	if _, ok := s.Chunks.GroundStack(tile); ok {
		t.Fatal("the oxygen is still lying outdoors, far above where it condenses")
	}
	if got, want := airAt(t, s, tile), planet(t, data.Planet_Earthlike).Atmosphere; got != want {
		t.Fatalf("the air outdoors is %v, want the planet's %v", got, want)
	}
}

func TestAGasesItemColderThanItCondensesStaysPut(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	s.Chunks.SetPlanet(constant(t, -200, 1))
	tile := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 1, anywhere)
	s.Chunks.SetGroundStack(tile, ecs.GroundStack{Kind: data.Item_Oxygen, Amount: 3 * data.MilligramsPerKg})
	s.Tick(2 * data.TPS)
	if on, _ := s.Chunks.GroundStack(tile); on.Amount != 3*kg {
		t.Fatalf("%d mg of the 3 kg of oxygen are left at -200 °C, want it all", on.Amount)
	}
}
