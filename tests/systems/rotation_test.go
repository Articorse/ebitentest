package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

// A thin bar sticking 30px out to the right of its entity, which is turned and scaled
func addBar(t *testing.T, x *ecs.ECSContainer, rotation, scale float64) common.EntityId {
	t.Helper()
	bar, err := shapes.NewPolygonShape([]utils.Vec2f{{X: 0, Y: -1}, {X: 30, Y: -1}, {X: 30, Y: 1}, {X: 0, Y: 1}}, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	return testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, scale, rotation),
		ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, ecs.Layer_Enemy, bar))
}

func addTargetAt(t *testing.T, x *ecs.ECSContainer, pos utils.Vec2f) common.EntityId {
	t.Helper()
	e := addTarget(t, x, 0)
	testutil.SetPos(t, x, e, pos)
	return e
}

// Through the broad phase and the shapes, as the game checks hurtboxes against hitboxes
func hits(x *ecs.ECSContainer, a common.EntityId, targets ...common.EntityId) []common.EntityId {
	pairs := []common.EntityPair{}
	for _, b := range targets {
		pairs = append(pairs, common.EntityPair{A: a, B: b})
	}
	hurtboxes, hitboxes := ecs.Colliders(&x.HurtboxColliders), ecs.Colliders(&x.HitboxColliders)
	hit := []common.EntityId{}
	for _, c := range collisionsystem.GetCollisions(hurtboxes, hitboxes, collisionsystem.GetDirectedAABBCollisions(hurtboxes, hitboxes, pairs, x), x) {
		hit = append(hit, c.B)
	}
	return hit
}

func TestCollidersTurnWithTheirEntity(t *testing.T) {
	x := testutil.NewECS(t)
	bar := addBar(t, x, math.Pi/2, 1)
	below := addTargetAt(t, x, utils.Vec2f{Y: 28})
	right := addTargetAt(t, x, utils.Vec2f{X: 28})

	if got := hits(x, bar, below, right); len(got) != 1 || got[0] != below {
		t.Fatalf("a bar turned to point down hit %v, want only the target below it", got)
	}
}

func TestCollidersTurnWithWhatTheyreAttachedTo(t *testing.T) {
	x := testutil.NewECS(t)
	holder := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	bar := addBar(t, x, 0, 1)
	testutil.Attach(t, x, bar, holder, ecs.Link_Held)
	tr, _ := x.Transforms.Get(holder)
	tr.SetLocalRotation(math.Pi)
	left := addTargetAt(t, x, utils.Vec2f{X: -28})
	right := addTargetAt(t, x, utils.Vec2f{X: 28})

	if got := hits(x, bar, left, right); len(got) != 1 || got[0] != left {
		t.Fatalf("a bar held by something turned around hit %v, want only the target to the left", got)
	}
}

func TestCollidersScaleWithTheirEntity(t *testing.T) {
	x := testutil.NewECS(t)
	bar := addBar(t, x, 0, 2)
	far := addTargetAt(t, x, utils.Vec2f{X: 58})

	if got := hits(x, bar, far); len(got) != 1 {
		t.Fatal("a bar scaled to 60px didn't reach a target 58px away")
	}
}

func TestTheAABBOfATurnedColliderCoversWhereItIs(t *testing.T) {
	x := testutil.NewECS(t)
	bar := addBar(t, x, math.Pi/2, 2)
	col, _ := x.HurtboxColliders.Get(bar)

	aabb, err := x.ColliderWorldAABB(bar, col)
	if err != nil {
		t.Fatal(err)
	}
	want := [2]utils.Vec2f{{X: -2}, {X: 2, Y: 60}}
	if !testutil.ApproxEqual(aabb[0], want[0]) || !testutil.ApproxEqual(aabb[1], want[1]) {
		t.Fatalf("the AABB of a bar turned down and doubled is %v, want %v", aabb, want)
	}
}

func TestATurnedRectangleOnlyMovesItsOffset(t *testing.T) {
	x := testutil.NewECS(t)
	box, err := shapes.NewRectangleShape(10, 2, utils.Vec2f{X: 20})
	if err != nil {
		t.Fatal(err)
	}
	e := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, math.Pi/2),
		ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, ecs.Layer_Enemy, box))
	// Reached by the box lying across, but not by one standing upright
	across := addTargetAt(t, x, utils.Vec2f{X: 8, Y: 20})
	along := addTargetAt(t, x, utils.Vec2f{Y: 28})

	if got := hits(x, e, across, along); len(got) != 1 || got[0] != across {
		t.Fatalf("a turned box hit %v, want only the target beside its turned offset", got)
	}
}

func TestATurnedColliderOccupiesTheTileItsTurnedOnto(t *testing.T) {
	x := testutil.NewECS(t)
	box, err := shapes.NewRectangleShape(4, 4, utils.Vec2f{X: data.TileSize})
	if err != nil {
		t.Fatal(err)
	}
	testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, math.Pi/2),
		ecs.NewPhysicsColliderComponent(ecs.Layer_Player, ecs.Layer_Wall, ecs.Collider_Mob, box))

	if !x.IsTileOccupied(utils.Vec2i{Y: 1}) || x.IsTileOccupied(utils.Vec2i{X: 1}) {
		t.Fatal("a collider turned from the tile to the right onto the one below doesn't occupy that one instead")
	}
}
