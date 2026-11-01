package systems_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/damagesystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestSingleTickContactDamageHitsEveryTargetOnceThenDisablesItsHurtbox(t *testing.T) {
	x := testutil.NewECS(t)
	shape, err := shapes.NewCircleShape(5, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	explosion := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0),
		ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, ecs.Layer_Enemy, shape),
		ecs.NewContactDamageComponent(-1, 0, false, true, 3))
	targets := []common.EntityId{}
	for range 2 {
		targets = append(targets, testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewHitpointsComponent(10, 0),
			ecs.NewHitboxColliderComponent(ecs.Layer_Enemy, ecs.Layer_FriendlyProjectile, shape)))
	}

	collisions := []common.Collision{}
	for _, e := range targets {
		collisions = append(collisions, common.Collision{A: explosion, B: e, Vector: utils.Vec2f{X: 1}})
	}
	if _, err := damagesystem.DealContactDamage(collisions, nil, noTiles{}, x); err != nil {
		t.Fatal(err)
	}

	for _, e := range targets {
		hp, _ := x.Hitpoints.Get(e)
		if hp.Current() != 7 {
			t.Errorf("target %d has %d hitpoints, want 7", e, hp.Current())
		}
		hitbox, _ := x.HitboxColliders.Get(e)
		if !hitbox.Enabled() {
			t.Errorf("target %d had its hitbox disabled", e)
		}
	}
	hurtbox, _ := x.HurtboxColliders.Get(explosion)
	if hurtbox.Enabled() {
		t.Error("the explosion's hurtbox is still enabled after its single tick")
	}
}

type noTiles struct{}

func (noTiles) DamageTile(utils.Vec2i, int) bool { return false }
