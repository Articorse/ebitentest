package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestTheBodyStatusListsPartsThenTheCapacitiesTheyGovern(t *testing.T) {
	x := testutil.NewECS(t)
	x.SetBodyAtlas(testutil.BodyAtlasFrom(t, `{
		"capacities": [{"id": 2, "name": "Seeing"}, {"id": 1, "name": "Moving", "floorPercent": 15}, {"id": 3, "name": "Hearing"}],
		"bodies": [{"id": 1, "name": "Legs", "parts": [
			{"name": "torso", "maxHealth": 20, "coverage": 0, "vital": true},
			{"name": "left leg", "maxHealth": 15, "coverage": 100, "capacities": [1]},
			{"name": "eye", "maxHealth": 3, "coverage": 0, "capacities": [2]}]}]}`))
	e := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewBodyComponent(data.Body_Humanoid))
	if _, err := x.Damage(e, 4); err != nil {
		t.Fatal(err)
	}

	got, err := drawsystem.BodyStatusText(e, x)
	if err != nil {
		t.Fatal(err)
	}
	want := "torso 20/20\nleft leg 11/15\neye 3/3\n\nMoving 73%\nSeeing 100%"
	if got != want {
		t.Fatalf("the body status is\n%s\nwant\n%s", got, want)
	}

	if _, err := drawsystem.BodyStatusText(testutil.AddMovable(t, x, utils.Vec2f{}, false), x); err == nil {
		t.Fatal("an entity without a body has a status")
	}
}
