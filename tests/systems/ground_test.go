package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

// Walked at the speeds told, and at full speed elsewhere
type groundSpeeds struct {
	ecs.Terrain
	speeds map[utils.Vec2i]int
}

func (g groundSpeeds) WalkSpeedPercent(t utils.Vec2i) int {
	if s, ok := g.speeds[t]; ok {
		return s
	}
	return 100
}

func TestWalkingIsSlowedByTheGroundUnderfoot(t *testing.T) {
	x := testutil.NewECS(t)
	shallow, trunk := utils.Vec2i{X: 3}, utils.Vec2i{X: 6}
	x.SetTerrain(groundSpeeds{Terrain: x.Terrain(), speeds: map[utils.Vec2i]int{shallow: 70, trunk: 0}})
	walker := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	speedOn := func(tile utils.Vec2i) float64 {
		t.Helper()
		if err := x.SetWorldPos(walker, utils.TilePosToWorldPos(tile)); err != nil {
			t.Fatal(err)
		}
		vel, _ := x.Velocities.Get(walker)
		vel.SetLocalVector(utils.Vec2f{})
		press(t, x, walker, ecs.InputState{Analog1X: 1})
		return vel.LocalVector().X
	}

	full := speedOn(utils.Vec2i{})
	if got := speedOn(shallow); math.Abs(got-0.7*full) > 1e-9 {
		t.Fatalf("on ground walked at 70%%, it sped up by %v, want %v", got, 0.7*full)
	}
	if got := speedOn(trunk); got != full {
		t.Fatalf("on the walkable part of a tile that can't be walked through, it sped up by %v, want the full %v", got, full)
	}

	x.SetBodyAtlas(testutil.BodyAtlasFrom(t, `{
		"capacities": [{"id": 1, "name": "Moving", "floorPercent": 15}],
		"bodies": [{"id": 1, "name": "Legs", "parts": [
			{"name": "torso", "maxHealth": 20, "coverage": 0, "vital": true},
			{"name": "legs", "maxHealth": 10, "coverage": 100, "capacities": [1]}]}]}`))
	x.AddComponent(walker, ecs.NewBodyComponent(data.Body_Humanoid))
	if _, err := x.Damage(walker, 5); err != nil {
		t.Fatal(err)
	}
	if got := speedOn(shallow); math.Abs(got-0.5*0.7*full) > 1e-9 {
		t.Fatalf("with hurt legs on ground walked at 70%%, it sped up by %v, want both slowdowns, %v", got, 0.5*0.7*full)
	}
}
