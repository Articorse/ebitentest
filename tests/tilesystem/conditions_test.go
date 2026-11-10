package tilesystem_test

import (
	"bytes"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"encoding/gob"
	"strings"
	"testing"
)

// Outdoors on a planet whose air is the oxygen and carbon dioxide given, in grams per tile, at celsius, whatever the
// time and place, with something of the body standing in it
func outdoorsIn(t *testing.T, oxygen, carbonDioxide int64, celsius float64, kind data.BodyEnum) (*testutil.Session, common.EntityId) {
	t.Helper()
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	p := constant(t, celsius, 1)
	p.Atmosphere[data.Gas_Oxygen] = oxygen * data.MilligramsPerGram
	p.Atmosphere[data.Gas_CarbonDioxide] = carbonDioxide * data.MilligramsPerGram
	s.Chunks.SetPlanet(p)
	tile := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 1, anywhere)
	return s, breather(t, s, tile, kind)
}

func severity(s *testutil.Session, e common.EntityId, name string) int {
	for _, c := range s.ECS.Conditions(e) {
		if c.Name == name {
			return c.Percent
		}
	}
	return 0
}

func TestAConditionGetsWorseFasterTheDeeperIntoWhatsBadForIt(t *testing.T) {
	for _, c := range []struct {
		name        string
		condition   string
		oxygen, co2 int64
		celsius     float64
		after       int
		want        int
	}{
		{"just short of hypoxia's start", "Hypoxia", 400, 0, 15, 60, 0},
		{"halfway from where hypoxia starts to a minute to full", "Hypoxia", 250, 0, 15, 30, 25},
		{"where hypoxia is full in a minute", "Hypoxia", 100, 0, 15, 30, 50},
		{"where hypoxia is full in 10 seconds", "Hypoxia", 0, 0, 15, 5, 50},
		{"where CO2 poisoning is full in 3 minutes", "CO2 poisoning", 900, 500, 15, 90, 50},
		{"where CO2 poisoning is full in 30 seconds", "CO2 poisoning", 900, 1000, 15, 15, 50},
		{"past CO2 poisoning's last point", "CO2 poisoning", 900, 2000, 15, 15, 50},
		{"where hypothermia is full in 10 minutes", "Hypothermia", 900, 100, -10, 300, 50},
		{"where heatstroke is full in a minute", "Heatstroke", 900, 100, 70, 30, 50},
		{"where decompression is full in 15 seconds", "Decompression", 0, 0, 15, 7, 46},
		{"where barotrauma is full in 5 minutes", "Barotrauma", 4900, 100, 15, 150, 50},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, e := outdoorsIn(t, c.oxygen, c.co2, c.celsius, data.Body_Giant)
			s.Tick(c.after * data.TPS)
			if got := severity(s, e, c.condition); abs(got-c.want) > 1 {
				t.Fatalf("after %d seconds, %s is at %d%%, want %d%%", c.after, c.condition, got, c.want)
			}
		})
	}
}

func TestAConditionGetsBetterOnceItsCauseIsGone(t *testing.T) {
	s, e := outdoorsIn(t, 0, 0, 15, data.Body_Giant)
	s.Tick(5 * data.TPS)
	if !s.ECS.Worsening(e) {
		t.Fatal("hypoxia isn't getting worse with no oxygen")
	}
	s.Chunks.SetPlanet(constant(t, 15, 1))
	s.Tick(60 * data.TPS)
	if got := severity(s, e, "Hypoxia"); got != 0 {
		t.Fatalf("a minute in good air after reaching 50%%, hypoxia is at %d%%, want gone", got)
	}
	if s.ECS.Worsening(e) {
		t.Fatal("it's worsening in good air")
	}
}

func TestAConditionDoesNoDamageBelowHalfAndKillsByAVitalPartAtFull(t *testing.T) {
	s, e := outdoorsIn(t, 0, 0, 15, data.Body_Humanoid)
	s.Tick(4 * data.TPS)
	if health, _, _ := s.ECS.PartHealth(e, "lungs"); health != 15 {
		t.Fatalf("below half severity, the lungs are at %d of 15", health)
	}
	s.Tick(6 * data.TPS)
	if health, _, _ := s.ECS.PartHealth(e, "lungs"); health == 15 {
		t.Fatal("at full severity, the lungs aren't hurt")
	}
	for range 20 * data.TPS {
		s.Tick(1)
		if !s.ECS.HasEntity(e) {
			return
		}
	}
	t.Fatal("with its lungs given 20 seconds at full hypoxia, it's still alive")
}

func TestHighSeverityTakesCapabilityAndWithItEveryOtherCapacityDown(t *testing.T) {
	s, e := outdoorsIn(t, 900, 100, -40, data.Body_Giant)
	// 80% hypothermia is halfway from where capability starts going to where it's gone
	s.Tick(48 * data.TPS)
	if got := s.ECS.Capacity(e, data.Capacity_Capability); abs(got-data.FullEfficiency/2) > 40 {
		t.Fatalf("at %d%% hypothermia, capability is %d of %d, want about half", severity(s, e, "Hypothermia"), got, data.FullEfficiency)
	}
	if got := s.ECS.Capacity(e, data.Capacity_Moving); got > data.FullEfficiency/2+40 {
		t.Fatalf("with half its capability, it can move at %d of %d", got, data.FullEfficiency)
	}
}

func TestWhatDoesntBreatheStillFeelsTheColdAndThePressure(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	s.ECS.SetBodyAtlas(testutil.BodyAtlasFrom(t, `{
		"capacities": [{"id": 3, "name": "Capability"}],
		"bodies": [{"id": 1, "name": "Stone", "parts": [
			{"name": "torso", "maxHealth": 20, "coverage": 100, "vital": true},
			{"name": "lungs", "maxHealth": 20, "coverage": 0}]}]}`))
	s.Chunks.SetPlanet(constant(t, -40, 0))
	tile := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 1, anywhere)
	e := breather(t, s, tile, data.Body_Humanoid)
	s.Tick(5 * data.TPS)
	for _, want := range []string{"Hypothermia", "Decompression"} {
		if severity(s, e, want) == 0 {
			t.Errorf("in a cold vacuum, what doesn't breathe has no %s", want)
		}
	}
	if severity(s, e, "Hypoxia") != 0 {
		t.Error("what doesn't breathe has hypoxia")
	}
}

func TestConditionsAreShownAndSaved(t *testing.T) {
	s, e := outdoorsIn(t, 0, 0, 15, data.Body_Giant)
	s.Tick(5 * data.TPS)
	status, err := drawsystem.BodyStatusText(e, s.ECS)
	if err != nil || !strings.Contains(status, "Hypoxia 50% (lungs), worsening") {
		t.Fatalf("the body status is\n%s\nwithout the hypoxia (%v)", status, err)
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(s.ECS.GetEntityComponents(e)); err != nil {
		t.Fatal(err)
	}
	var dtos []ecs.ComponentDto
	if err := gob.NewDecoder(&buf).Decode(&dtos); err != nil {
		t.Fatal(err)
	}
	x := testutil.NewECS(t)
	comps := []ecs.Component{}
	for _, d := range dtos {
		c, err := ecs.DtoToComponent(d)
		if err != nil {
			t.Fatal(err)
		}
		comps = append(comps, c)
	}
	loaded := testutil.Add(t, x, comps...)
	if got, want := x.Conditions(loaded), s.ECS.Conditions(e); len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("loaded, the conditions are %v, want %v", got, want)
	}
	if got, want := x.Capacity(loaded, data.Capacity_Capability), s.ECS.Capacity(e, data.Capacity_Capability); got != want {
		t.Fatalf("loaded, its capability is %d, want %d", got, want)
	}
}
