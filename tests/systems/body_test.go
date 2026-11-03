package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestHurtLegsSlowMovementDownToTheFloor(t *testing.T) {
	x := testutil.NewECS(t)
	x.SetBodyAtlas(testutil.BodyAtlasFrom(t, `{
		"capacities": [{"id": 1, "name": "Moving", "floorPercent": 15}],
		"bodies": [{"id": 1, "name": "Legs", "parts": [
			{"name": "torso", "maxHealth": 20, "coverage": 0, "vital": true},
			{"name": "left leg", "maxHealth": 10, "coverage": 50, "capacities": [1]},
			{"name": "right leg", "maxHealth": 10, "coverage": 50, "capacities": [1]}]}]}`))
	walker := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewBodyComponent(data.Body_Humanoid))
	speed := func() float64 {
		t.Helper()
		vel, _ := x.Velocities.Get(walker)
		vel.SetLocalVector(utils.Vec2f{})
		press(t, x, walker, ecs.InputState{Analog1X: 1})
		return vel.LocalVector().X
	}

	full := speed()
	if full <= 0 {
		t.Fatal("an unhurt walker didn't speed up")
	}
	for {
		if left, _, _ := x.PartHealth(walker, "left leg"); left == 0 {
			break
		}
		if _, err := x.Damage(walker, 10); err != nil {
			t.Fatal(err)
		}
	}
	right, _, _ := x.PartHealth(walker, "right leg")
	want := full * float64((right*100)/2) / 1000
	if got := speed(); got != want {
		t.Fatalf("with a destroyed leg and the other at %d of 10, it sped up by %v, want %v", right, got, want)
	}

	for {
		if right, _, _ := x.PartHealth(walker, "right leg"); right == 0 {
			break
		}
		if _, err := x.Damage(walker, 10); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := speed(), full*0.15; got != want {
		t.Fatalf("without legs, it sped up by %v, want the floor of %v", got, want)
	}
}

func TestDamageToAVitalPartKills(t *testing.T) {
	for damage, wantDead := range map[int]bool{9: false, 10: true} {
		x := testutil.NewECS(t)
		source := addSource(t, x, ecs.NewContactDamageComponent(-1, 0, damage))
		target := testutil.AddMovable(t, x, utils.Vec2f{}, false, testutil.DummyBody(t, x, 10),
			ecs.NewHitboxColliderComponent(ecs.Layer_Enemy, ecs.Layer_FriendlyProjectile, newShape(t, "circle", utils.Vec2f{})))

		touchEntities(t, x, []common.EntityId{source}, []common.EntityId{target})
		if dead := !x.HasEntity(target); dead != wantDead {
			t.Errorf("a hit of %d on its only, vital part of 10 killed it %v, want %v", damage, dead, wantDead)
		}
	}
}
