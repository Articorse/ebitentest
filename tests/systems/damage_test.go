package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/damagesystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func addSource(t *testing.T, x *ecs.ECSContainer, cd *ecs.ContactDamage) common.EntityId {
	t.Helper()
	shape, err := shapes.NewCircleShape(5, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	return testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0),
		ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, ecs.Layer_Enemy, shape), cd)
}

func addTarget(t *testing.T, x *ecs.ECSContainer, invulMs int) common.EntityId {
	t.Helper()
	shape, err := shapes.NewCircleShape(5, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	return testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewHitpointsComponent(100, invulMs),
		ecs.NewHitboxColliderComponent(ecs.Layer_Enemy, ecs.Layer_FriendlyProjectile, shape))
}

// One tick of contact damage, with every source touching every target
func touchEntities(t *testing.T, x *ecs.ECSContainer, sources, targets []common.EntityId) {
	t.Helper()
	collisions := []common.Collision{}
	for _, s := range sources {
		for _, e := range targets {
			collisions = append(collisions, common.Collision{A: s, B: e, Vector: utils.Vec2f{X: 1}})
		}
	}
	if _, err := damagesystem.DealContactDamage(collisions, nil, x); err != nil {
		t.Fatal(err)
	}
	testutil.RemoveScheduled(t, x)
}

func lost(x *ecs.ECSContainer, e common.EntityId) int {
	hp, _ := x.Hitpoints.Get(e)
	return 100 - hp.Current()
}

func TestContactDamageHitsEachEntityOnceByDefault(t *testing.T) {
	x := testutil.NewECS(t)
	source := addSource(t, x, ecs.NewContactDamageComponent(-1, 0, 3))
	target := addTarget(t, x, 0)

	for range 10 {
		touchEntities(t, x, []common.EntityId{source}, []common.EntityId{target})
	}

	if lost(x, target) != 3 {
		t.Fatalf("target lost %d hitpoints over 10 ticks of touching, want 3: hit once", lost(x, target))
	}
}

func TestContactDamageHitsAgainEveryRehitInterval(t *testing.T) {
	x := testutil.NewECS(t)
	source := addSource(t, x, ecs.NewContactDamageComponent(-1, 0, 1).WithRehitIntervalMs(3*data.TickMs))
	target := addTarget(t, x, 0)

	for range 7 {
		touchEntities(t, x, []common.EntityId{source}, []common.EntityId{target})
	}

	if lost(x, target) != 3 {
		t.Fatalf("target lost %d hitpoints over 7 ticks, want 3: hit at ticks 0, 3 and 6", lost(x, target))
	}
}

func TestContactDamageOnlyHurtsWhileActive(t *testing.T) {
	x := testutil.NewECS(t)
	explosion := addSource(t, x, ecs.NewContactDamageComponent(-1, 0, 3).WithActiveMs(data.TickMs))
	first, second := addTarget(t, x, 0), addTarget(t, x, 0)
	late := addTarget(t, x, 0)

	touchEntities(t, x, []common.EntityId{explosion}, []common.EntityId{first, second})
	touchEntities(t, x, []common.EntityId{explosion}, []common.EntityId{first, second, late})

	if lost(x, first) != 3 || lost(x, second) != 3 {
		t.Fatalf("targets in the first tick lost %d and %d, want 3 each", lost(x, first), lost(x, second))
	}
	if lost(x, late) != 0 {
		t.Fatal("a target arriving after the first tick was hit")
	}
}

func TestDyingOnContactUsesUpTheSource(t *testing.T) {
	x := testutil.NewECS(t)
	bullet := addSource(t, x, ecs.NewContactDamageComponent(-1, 0, 3).WithDieOnContact())
	target := addTarget(t, x, 0)

	touchEntities(t, x, []common.EntityId{bullet}, []common.EntityId{target})

	if x.HasEntity(bullet) || lost(x, target) != 3 {
		t.Fatalf("bullet still exists %v, target lost %d; want the bullet used up on its hit", x.HasEntity(bullet), lost(x, target))
	}
}

func TestInvulnerabilityStillProtectsFromOtherSources(t *testing.T) {
	x := testutil.NewECS(t)
	a := addSource(t, x, ecs.NewContactDamageComponent(-1, 0, 3))
	b := addSource(t, x, ecs.NewContactDamageComponent(-1, 0, 3))
	target := addTarget(t, x, 1000)

	touchEntities(t, x, []common.EntityId{a, b}, []common.EntityId{target})

	if lost(x, target) != 3 {
		t.Fatalf("target lost %d from two sources in one tick, want 3: the first hit makes it invulnerable", lost(x, target))
	}
}

func TestSavedSourceRemembersWhatItHit(t *testing.T) {
	x := testutil.NewECS(t)
	source := addSource(t, x, ecs.NewContactDamageComponent(-1, 0, 3))
	target := addTarget(t, x, 0)
	touchEntities(t, x, []common.EntityId{source}, []common.EntityId{target})

	members, err := x.GetCompositionComponents(source)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)

	cd, _ := y.ContactDamages.Get(source)
	if cd.CanHitEntity(target) {
		t.Fatal("after saving, the source could hit its target again")
	}
}
