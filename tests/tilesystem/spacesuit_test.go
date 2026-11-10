package tilesystem_test

import (
	"ebittest/climate"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"strings"
	"testing"
)

// Into the hands of something of the body that does nothing but breathe, holding the oxygen and carbon dioxide given in
// kilograms, and worn
func suitUp(t *testing.T, s *testutil.Session, e common.EntityId, oxygenKg, carbonDioxideKg int64) common.EntityId {
	t.Helper()
	s.ECS.AddComponent(e, ecs.NewInventoryComponent(1, 1, data.PlayerMaxCarryMg))
	s.ECS.AddComponent(e, ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{}))
	air := climate.Air{data.Gas_Oxygen: oxygenKg * data.MilligramsPerKg, data.Gas_CarbonDioxide: carbonDioxideKg * data.MilligramsPerKg}
	suit := testutil.Add(t, s.ECS,
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(testutil.WorldPos(t, s.ECS, e), 1, 0),
		ecs.NewWornEquipmentComponent(ecs.Equipable_Torso, []data.BodyEnum{data.Body_Humanoid}, ""),
		ecs.NewNamedItemComponent("Spacesuit", 10*data.MilligramsPerKg),
		ecs.NewSealedAirComponent(air, data.SpacesuitAirTiles),
	)
	if err := s.ECS.PutInInventory(e, 0, suit); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.ECS.Wear(e, suit, ecs.Equip_Torso); err != nil || !ok {
		t.Fatalf("the spacesuit couldn't be worn (%v)", err)
	}
	return suit
}

func TestTheWearerOfASpacesuitBreathesItsAirRatherThanTheRooms(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := sealedRoom(t, s, 3)
	before := airAt(t, s, middle)
	e := breather(t, s, middle, data.Body_Humanoid)
	suit := suitUp(t, s, e, 10, 0)

	s.Tick(60 * data.TPS)
	if got := airAt(t, s, middle); got != before {
		t.Fatalf("a minute's breathing in a spacesuit changed the room's air from %v to %v", before, got)
	}
	sealed, _ := s.ECS.SealedAirs.Get(suit)
	breathed := int64(13889 * data.AirStepTicks / data.TPS * 60 * data.TPS / data.AirStepTicks)
	if got := sealed.Contents(); got[data.Gas_Oxygen] != 10*data.MilligramsPerKg-breathed || got[data.Gas_CarbonDioxide] != breathed {
		t.Fatalf("after a minute, the spacesuit holds %v, want %d mg of its oxygen breathed into carbon dioxide", got, breathed)
	}
	if got, _ := s.ECS.AirOf(e); got != sealed.PerTile() {
		t.Fatalf("the wearer breathes %v, want the spacesuit's %v per tile", got, sealed.PerTile())
	}
}

func TestASpacesuitKeepsOutTheVacuumAndTheCold(t *testing.T) {
	s, e := outdoorsIn(t, 0, 0, -40, data.Body_Humanoid)
	suitUp(t, s, e, 10, 0)
	s.Tick(60 * data.TPS)
	if conditions := s.ECS.Conditions(e); len(conditions) != 0 {
		t.Fatalf("a minute in a cold vacuum in a full spacesuit gave %v", conditions)
	}
	if s.ECS.InBadAir(e) {
		t.Fatal("the full spacesuit's air is bad to breathe")
	}
	if got, _ := s.ECS.AirTemperatureOf(e); got.Celsius() != data.SealedAirCelsius {
		t.Fatalf("inside the spacesuit it's %.1f °C", got.Celsius())
	}
}

func TestASpacesuitFullOfCarbonDioxideIsBadToBreatheInGoodAir(t *testing.T) {
	s, e := outdoorsIn(t, 900, 100, 15, data.Body_Humanoid)
	suitUp(t, s, e, 5, 5)
	if !s.ECS.InBadAir(e) {
		t.Fatal("a spacesuit half full of carbon dioxide isn't bad to breathe")
	}
	s.Tick(30 * data.TPS)
	if severity(s, e, "CO2 poisoning") == 0 {
		t.Fatal("half a minute in a spacesuit half full of carbon dioxide didn't poison its wearer")
	}
}

func TestTakingASpacesuitOffBreathesTheAirAroundAgain(t *testing.T) {
	s, e := outdoorsIn(t, 0, 0, -40, data.Body_Humanoid)
	suitUp(t, s, e, 10, 0)
	if err := s.ECS.TakeOff(e, ecs.Equip_Torso); err != nil {
		t.Fatal(err)
	}
	if !s.ECS.InBadAir(e) {
		t.Fatal("out of the spacesuit, the vacuum isn't bad to breathe")
	}
	if got, _ := s.ECS.AirTemperatureOf(e); got.Celsius() != -40 {
		t.Fatalf("out of the spacesuit, it's %.1f °C", got.Celsius())
	}
}

func TestWhatsInASpacesuitIsShown(t *testing.T) {
	s, e := outdoorsIn(t, 900, 100, 15, data.Body_Humanoid)
	suitUp(t, s, e, 8, 2)
	want := "Holding 8kg O2, 2kg CO2"
	if status, err := drawsystem.BodyStatusText(e, s.ECS); err != nil || !strings.Contains(status, want) {
		t.Fatalf("the body status is\n%s\nwithout %q (%v)", status, want, err)
	}
	if got := drawsystem.EquipSlotTooltip(e, ecs.Equip_Torso, s.ECS); len(got) != 3 || got[2] != want {
		t.Fatalf("the torso's tooltip is %v, want it to end with %q", got, want)
	}
}

func TestASpacesuitsAirIsSaved(t *testing.T) {
	s, e := outdoorsIn(t, 900, 100, 15, data.Body_Humanoid)
	suit := suitUp(t, s, e, 7, 3)
	sealed, _ := s.ECS.SealedAirs.Get(suit)
	for _, d := range s.ECS.GetEntityComponents(suit) {
		c, err := ecs.DtoToComponent(d)
		if err != nil {
			t.Fatal(err)
		}
		if loaded, ok := c.(*ecs.SealedAir); ok {
			if loaded.Contents() != sealed.Contents() || loaded.PerTile() != sealed.PerTile() {
				t.Fatalf("loaded, the spacesuit holds %v, want %v", loaded.Contents(), sealed.Contents())
			}
			return
		}
	}
	t.Fatal("the spacesuit was saved without its air")
}
