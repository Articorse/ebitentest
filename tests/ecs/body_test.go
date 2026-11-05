package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/visionsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"os"
	"path/filepath"
	"testing"
)

// Capacity 1 is Moving, as in the game's atlas
const legsJson = `{
	"capacities": [{"id": 1, "name": "Moving", "floorPercent": 15}, {"id": 2, "name": "Pumping", "floorPercent": 0}],
	"bodies": [
		{"id": 1, "name": "Legs", "parts": [
			{"name": "torso", "maxHealth": 20, "coverage": 0, "vital": true},
			{"name": "left leg", "maxHealth": 15, "coverage": 50, "capacities": [1]},
			{"name": "right leg", "maxHealth": 15, "coverage": 50, "capacities": [1]}]},
		{"id": 2, "name": "Weighted", "parts": [
			{"name": "big", "maxHealth": 10000, "coverage": 60},
			{"name": "left", "maxHealth": 10000, "coverage": 20},
			{"name": "right", "maxHealth": 10000, "coverage": 20}]},
		{"id": 3, "name": "Brittle", "parts": [
			{"name": "shell", "maxHealth": 1, "coverage": 100},
			{"name": "core", "maxHealth": 100, "coverage": 1}]},
		{"id": 4, "name": "Hearted", "parts": [
			{"name": "heart", "maxHealth": 5, "coverage": 100, "capacities": [2]},
			{"name": "skin", "maxHealth": 1000, "coverage": 0}],
			"vitalCapacities": [2]}
	]
}`

func withBody(t *testing.T, x *ecs.ECSContainer, kind data.BodyEnum) common.EntityId {
	t.Helper()
	return testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewBodyComponent(kind))
}

func testBodies(t *testing.T) *ecs.ECSContainer {
	t.Helper()
	x := testutil.NewECS(t)
	x.SetBodyAtlas(testutil.BodyAtlasFrom(t, legsJson))
	return x
}

func damage(t *testing.T, x *ecs.ECSContainer, e common.EntityId, amount int) bool {
	t.Helper()
	dead, err := x.Damage(e, amount)
	if err != nil {
		t.Fatal(err)
	}
	return dead
}

func TestTheBodyAtlasRefusesBrokenBodies(t *testing.T) {
	part := `{"name": "torso", "maxHealth": 20, "coverage": 100}`
	body := func(parts string) string {
		return `{"capacities": [{"id": 1, "name": "Moving"}], "bodies": [{"id": 1, "name": "Body", "parts": [` + parts + `]}]}`
	}
	for _, c := range []struct {
		name string
		json string
	}{
		{"capacity 0", `{"capacities": [{"id": 0, "name": "Moving"}]}`},
		{"a capacity id used twice", `{"capacities": [{"id": 1, "name": "A"}, {"id": 1, "name": "B"}]}`},
		{"a floor above 100%", `{"capacities": [{"id": 1, "name": "Moving", "floorPercent": 101}]}`},
		{"a negative floor", `{"capacities": [{"id": 1, "name": "Moving", "floorPercent": -1}]}`},
		{"body 0", `{"bodies": [{"id": 0, "name": "Body", "parts": [` + part + `]}]}`},
		{"a body id used twice", `{"bodies": [{"id": 1, "name": "A", "parts": [` + part + `]}, {"id": 1, "name": "B", "parts": [` + part + `]}]}`},
		{"a part without a name", body(`{"maxHealth": 20, "coverage": 100}`)},
		{"a part name used twice", body(part + `, ` + part)},
		{"a part without health", body(`{"name": "torso", "maxHealth": 0, "coverage": 100}`)},
		{"a negative coverage", body(part + `, {"name": "leg", "maxHealth": 20, "coverage": -1}`)},
		{"nothing to hit", body(`{"name": "torso", "maxHealth": 20, "coverage": 0}`)},
		{"no parts", body(``)},
		{"a part governing a capacity that isn't defined", body(`{"name": "torso", "maxHealth": 20, "coverage": 100, "capacities": [2]}`)},
		{"a vital capacity that isn't defined", `{"capacities": [{"id": 1, "name": "Moving"}], "bodies": [{"id": 1, "name": "Body", "parts": [` + part + `], "vitalCapacities": [2]}]}`},
	} {
		path := filepath.Join(t.TempDir(), "bodies.json")
		if err := os.WriteFile(path, []byte(c.json), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ecs.LoadBodyAtlasFromJson(path); err == nil {
			t.Errorf("an atlas with %s was accepted", c.name)
		}
	}
}

func TestTheHumanoidHasAVitalTorsoAndTwoLegs(t *testing.T) {
	x := testutil.NewECS(t)
	e := withBody(t, x, data.Body_Humanoid)

	for part, want := range map[string]int{"torso": 20, "left leg": 15, "right leg": 15} {
		if health, maxHealth, ok := x.PartHealth(e, part); !ok || health != want || maxHealth != want {
			t.Errorf("the humanoid's %s has %d of %d health (%v), want %d", part, health, maxHealth, ok, want)
		}
	}
	if got := x.Capacity(e, data.Capacity_Moving); got != data.FullEfficiency {
		t.Fatalf("an unhurt humanoid moves at %d, want full efficiency", got)
	}

	for !damage(t, x, e, 1) {
	}
	if health, _, _ := x.PartHealth(e, "torso"); health != 0 {
		t.Fatalf("the humanoid died with its torso at %d", health)
	}
}

func TestHitsLandOnPartsByTheirCoverage(t *testing.T) {
	x := testBodies(t)
	e := withBody(t, x, 2)
	for range 2000 {
		damage(t, x, e, 1)
	}

	for part, want := range map[string]int{"big": 1200, "left": 400, "right": 400} {
		health, _, _ := x.PartHealth(e, part)
		if got := 10000 - health; got < want*9/10 || got > want*11/10 {
			t.Errorf("of 2000 hits, %d landed on %s, want about %d", got, part, want)
		}
	}
}

func TestDestroyedPartsAreNoLongerHitAndExcessDamageIsLost(t *testing.T) {
	x := testBodies(t)
	e := withBody(t, x, 3)
	for {
		if shell, _, _ := x.PartHealth(e, "shell"); shell == 0 {
			break
		}
		damage(t, x, e, 5)
	}

	core, _, _ := x.PartHealth(e, "core")
	if core%5 != 0 {
		t.Fatalf("the core has %d health after hits of 5", core)
	}
	for range 3 {
		damage(t, x, e, 5)
	}
	if got, _, _ := x.PartHealth(e, "core"); got != core-15 {
		t.Fatalf("with the shell destroyed, 3 hits of 5 left the core at %d, want %d", got, core-15)
	}
	if shell, _, _ := x.PartHealth(e, "shell"); shell != 0 {
		t.Fatalf("the destroyed shell has %d health after more hits, want 0", shell)
	}
}

func TestMovingIsTheLegsAverageEfficiencyButNeverBelowItsFloor(t *testing.T) {
	x := testBodies(t)
	e := withBody(t, x, 1)

	for {
		left, _, _ := x.PartHealth(e, "left leg")
		right, _, _ := x.PartHealth(e, "right leg")
		want := max((left*1000/15+right*1000/15)/2, 150)
		if got := x.Capacity(e, data.Capacity_Moving); got != want {
			t.Fatalf("with legs at %d and %d of 15, moving is %d, want %d", left, right, got, want)
		}
		if left == 0 && right == 0 {
			break
		}
		if damage(t, x, e, 4) {
			t.Fatal("losing legs killed the body")
		}
	}
}

func TestAVitalCapacityAtNothingKills(t *testing.T) {
	x := testBodies(t)
	e := withBody(t, x, 4)
	if damage(t, x, e, 4) {
		t.Fatal("a hurt heart killed the body")
	}
	if !damage(t, x, e, 1) {
		t.Fatal("pumping at nothing didn't kill the body")
	}
}

func TestCapacitiesNoPartGovernsAreAtFullEfficiency(t *testing.T) {
	x := testBodies(t)
	brittle := withBody(t, x, 3)
	bodiless := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	for _, e := range []common.EntityId{brittle, bodiless} {
		if got := x.Capacity(e, data.Capacity_Moving); got != data.FullEfficiency {
			t.Errorf("entity %d without legs moves at %d, want full efficiency", e, got)
		}
	}
}

func TestADamagedBodyIsSavedAndItsCapacitiesWorkedOutAgain(t *testing.T) {
	x := testBodies(t)
	e := withBody(t, x, 1)
	for {
		if left, _, _ := x.PartHealth(e, "left leg"); left == 0 {
			break
		}
		damage(t, x, e, 15)
	}
	right, _, _ := x.PartHealth(e, "right leg")
	moving := x.Capacity(e, data.Capacity_Moving)

	members, err := x.GetCompositionComponents(e)
	if err != nil {
		t.Fatal(err)
	}
	comps := []ecs.Component{}
	for _, dto := range members[e] {
		comp, err := ecs.DtoToComponent(dto)
		if err != nil {
			t.Fatal(err)
		}
		comps = append(comps, comp)
	}
	y := testBodies(t)
	if err := y.AddEntityWithId(e, comps...); err != nil {
		t.Fatal(err)
	}

	if left, _, _ := y.PartHealth(e, "left leg"); left != 0 {
		t.Fatalf("the destroyed left leg loaded with %d health", left)
	}
	if got, _, _ := y.PartHealth(e, "right leg"); got != right {
		t.Fatalf("the right leg loaded with %d health, want %d", got, right)
	}
	if got := y.Capacity(e, data.Capacity_Moving); got != moving || got == data.FullEfficiency {
		t.Fatalf("the loaded body moves at %d, want %d", got, moving)
	}
}

func TestHurtEyesShortenVisionAndDestroyedOnesBlind(t *testing.T) {
	x := testutil.NewECS(t)
	x.SetBodyAtlas(testutil.BodyAtlasFrom(t, `{
		"capacities": [{"id": 2, "name": "Sight", "floorPercent": 0}],
		"bodies": [{"id": 1, "name": "Eyes", "parts": [
			{"name": "torso", "maxHealth": 20, "coverage": 0, "vital": true},
			{"name": "eyes", "maxHealth": 10, "coverage": 100, "capacities": [2]}]}]}`))
	x.SetTerrain(skyTerrain{Terrain: x.Terrain(), roofFromX: 1000})
	viewer := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0),
		ecs.NewVisionComponent(40, 1), ecs.NewBodyComponent(data.Body_Humanoid))
	v, _ := x.Visions.Get(viewer)

	for _, c := range []struct{ eyes, rangeTiles int }{{10, 40}, {6, 24}, {1, 4}, {0, 0}} {
		for {
			if eyes, _, _ := x.PartHealth(viewer, "eyes"); eyes == c.eyes {
				break
			}
			damage(t, x, viewer, 1)
		}
		visionsystem.Tick(x)
		if _, radius := v.VisibleArea(); radius != c.rangeTiles {
			t.Errorf("with eyes at %d of 10, it sees %d tiles in daylight, want %d", c.eyes, radius, c.rangeTiles)
		}
	}
	if v.CanSee(utils.Vec2i{}) {
		t.Fatal("with destroyed eyes, it sees its own tile")
	}
}

func TestTheHumanoidHasEyesGoverningSight(t *testing.T) {
	x := testutil.NewECS(t)
	e := withBody(t, x, data.Body_Humanoid)
	if health, maxHealth, ok := x.PartHealth(e, "eyes"); !ok || health != 10 || maxHealth != 10 {
		t.Fatalf("the humanoid's eyes have %d of %d health (%v), want 10", health, maxHealth, ok)
	}
	_, capacities, err := x.BodyStatus(e)
	if err != nil {
		t.Fatal(err)
	}
	if len(capacities) != 2 || capacities[1].Name != "Sight" || capacities[1].Efficiency != data.FullEfficiency {
		t.Fatalf("the humanoid's capacities are %+v, want full Sight after Moving", capacities)
	}
}
