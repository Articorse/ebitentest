package systems_test

import (
	"cmp"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/ecs/systems/commonsystems"
	"ebittest/ecs/systems/damagesystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"slices"
	"testing"
)

func newShape(t *testing.T, kind string, offset utils.Vec2f) shapes.Shape {
	t.Helper()

	var s shapes.Shape
	var err error
	switch kind {
	case "rectangle":
		s, err = shapes.NewRectangleShape(10, 10, offset)
	case "circle":
		s, err = shapes.NewCircleShape(5, offset)
	case "polygon":
		s, err = shapes.NewPolygonShape([]utils.Vec2f{{X: -5, Y: -5}, {X: 5, Y: -5}, {X: 5, Y: 5}, {X: -5, Y: 5}}, offset)
	}
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func addBody(t *testing.T, x *ecs.ECSContainer, pos utils.Vec2f, shape shapes.Shape) common.EntityId {
	t.Helper()
	return testutil.AddMovable(t, x, pos, false, ecs.NewPhysicsColliderComponent(1, 1, ecs.Collider_Mob, shape))
}

func physicsCollisions(x *ecs.ECSContainer, pairs []common.EntityPair) []common.Collision {
	colliders := ecs.Colliders(&x.PhysicsColliders)
	return collisionsystem.GetCollisions(colliders, colliders, pairs, x)
}

func TestPushOutPointsFromAToBForEveryShapePair(t *testing.T) {
	kinds := []string{"rectangle", "circle", "polygon"}
	for _, aKind := range kinds {
		for _, bKind := range kinds {
			for _, moving := range []bool{false, true} {
				x := testutil.NewECS(t)
				a := addBody(t, x, utils.Vec2f{}, newShape(t, aKind, utils.Vec2f{}))
				b := addBody(t, x, utils.Vec2f{X: 8, Y: 1}, newShape(t, bKind, utils.Vec2f{}))
				if moving {
					v, _ := x.Velocities.Get(a)
					v.SetLocalVector(utils.Vec2f{X: 1})
				}

				cols := physicsCollisions(x, []common.EntityPair{{A: a, B: b}})
				if len(cols) != 1 {
					t.Errorf("%s-%s, moving %v: %d collisions, want 1", aKind, bKind, moving, len(cols))
					continue
				}
				if v := cols[0].Vector; v.X <= 0 || math.Abs(v.Y) >= v.X {
					t.Errorf("%s-%s, moving %v: push-out %v doesn't point from A to B", aKind, bKind, moving, v)
				}
			}
		}
	}
}

func TestCircleInsidePolygonLeavesThroughTheClosestEdge(t *testing.T) {
	x := testutil.NewECS(t)
	c := addBody(t, x, utils.Vec2f{}, newShape(t, "circle", utils.Vec2f{}))
	p := addBody(t, x, utils.Vec2f{X: 4}, newShape(t, "polygon", utils.Vec2f{}))

	cols := physicsCollisions(x, []common.EntityPair{{A: c, B: p}})
	if len(cols) != 1 {
		t.Fatalf("%d collisions, want 1", len(cols))
	}
	if want := (utils.Vec2f{X: 6}); !testutil.ApproxEqual(cols[0].Vector, want) {
		t.Fatalf("push-out %v, want %v: the radius plus the center's distance to the polygon's left edge", cols[0].Vector, want)
	}
}

func TestPolygonOffsetMovesWhereItCollides(t *testing.T) {
	x := testutil.NewECS(t)
	p := addBody(t, x, utils.Vec2f{}, newShape(t, "polygon", utils.Vec2f{X: 100}))
	atOffset := addBody(t, x, utils.Vec2f{X: 100}, newShape(t, "circle", utils.Vec2f{}))
	atEntity := addBody(t, x, utils.Vec2f{}, newShape(t, "circle", utils.Vec2f{}))

	cols := physicsCollisions(x, []common.EntityPair{{A: p, B: atOffset}, {A: p, B: atEntity}})
	if len(cols) != 1 || cols[0].B != atOffset {
		t.Fatalf("got collisions %+v, want only the one with the circle at the polygon's offset", cols)
	}
}

func TestProximatePairsAreUniqueAndInIdOrder(t *testing.T) {
	x := testutil.NewECS(t)
	for i := range 60 {
		testutil.AddMovable(t, x, utils.Vec2f{X: float64(i*37%500 - 250), Y: float64(i*53%500 - 250)}, false)
	}

	grid, err := commonsystems.PopulateSpatialHashGrid(x, 200)
	if err != nil {
		t.Fatal(err)
	}
	pairs, err := commonsystems.GetSHGProximities(grid, x)
	if err != nil {
		t.Fatal(err)
	}

	if len(pairs) == 0 {
		t.Fatal("no proximate pairs")
	}
	for i, p := range pairs {
		if p.A >= p.B {
			t.Fatalf("pair %+v isn't lower ID first", p)
		}
		if i > 0 && cmp.Or(cmp.Compare(pairs[i-1].A, p.A), cmp.Compare(pairs[i-1].B, p.B)) >= 0 {
			t.Fatalf("pairs %+v and %+v are out of order or repeated", pairs[i-1], p)
		}
	}
}

func TestPhysicsResolutionIsRepeatable(t *testing.T) {
	run := func() []utils.Vec2f {
		x := testutil.NewECS(t)
		for i := range 40 {
			e := addBody(t, x, utils.Vec2f{X: float64(i%8) * 6, Y: float64(i/8) * 6}, newShape(t, "circle", utils.Vec2f{}))
			v, _ := x.Velocities.Get(e)
			v.SetLocalVector(utils.Vec2f{X: math.Sin(float64(i)), Y: math.Cos(float64(i))})
		}

		for range 5 {
			grid, err := commonsystems.PopulateSpatialHashGrid(x, 200)
			if err != nil {
				t.Fatal(err)
			}
			proximate, err := commonsystems.GetSHGProximities(grid, x)
			if err != nil {
				t.Fatal(err)
			}
			colliders := ecs.Colliders(&x.PhysicsColliders)
			cols := physicsCollisions(x, collisionsystem.GetAABBCollisions(colliders, proximate, x))
			if _, err := collisionsystem.ResolvePhysicsCollisions(cols, x); err != nil {
				t.Fatal(err)
			}
		}

		state := []utils.Vec2f{}
		for _, e := range x.PhysicsColliders.GetEntities() {
			v, _ := x.Velocities.Get(e)
			state = append(state, testutil.WorldPos(t, x, e), v.LocalVector())
		}
		return state
	}

	want := run()
	for range 10 {
		if got := run(); !slices.Equal(got, want) {
			t.Fatal("the same collisions resolved to different positions or velocities")
		}
	}
}

func TestEntitiesWithHurtboxesAndHitboxesDamageEachOther(t *testing.T) {
	x := testutil.NewECS(t)
	fighters := []common.EntityId{}
	for _, pos := range []utils.Vec2f{{}, {X: 6}} {
		fighters = append(fighters, testutil.AddMovable(t, x, pos, false,
			ecs.NewHitpointsComponent(10, 0),
			ecs.NewContactDamageComponent(-1, 0, false, false, 3),
			ecs.NewHurtboxColliderComponent(ecs.Layer_Enemy, ecs.Layer_Enemy, newShape(t, "circle", utils.Vec2f{})),
			ecs.NewHitboxColliderComponent(ecs.Layer_Enemy, ecs.Layer_Enemy, newShape(t, "circle", utils.Vec2f{}))))
	}

	hurtboxes, hitboxes := ecs.Colliders(&x.HurtboxColliders), ecs.Colliders(&x.HitboxColliders)
	pairs := collisionsystem.GetDirectedAABBCollisions(hurtboxes, hitboxes, []common.EntityPair{{A: fighters[0], B: fighters[1]}}, x)
	cols := collisionsystem.GetCollisions(hurtboxes, hitboxes, pairs, x)
	if _, err := damagesystem.DealContactDamage(cols, nil, noTiles{}, x); err != nil {
		t.Fatal(err)
	}

	for _, e := range fighters {
		if hp, _ := x.Hitpoints.Get(e); hp.Current() != 7 {
			t.Errorf("fighter %d has %d hitpoints, want 7", e, hp.Current())
		}
	}
}

func TestCircleWithItsCenterInsideARectangleLeavesThroughTheNearestEdge(t *testing.T) {
	for _, moving := range []bool{false, true} {
		for _, c := range []struct {
			offset utils.Vec2f
			want   utils.Vec2f
		}{
			{offset: utils.Vec2f{X: 2}, want: utils.Vec2f{X: 8}},
			{offset: utils.Vec2f{X: -1, Y: 3}, want: utils.Vec2f{Y: 7}},
			{offset: utils.Vec2f{}, want: utils.Vec2f{X: -10}},
		} {
			x := testutil.NewECS(t)
			rect := addBody(t, x, utils.Vec2f{}, newShape(t, "rectangle", utils.Vec2f{}))
			circle := addBody(t, x, c.offset, newShape(t, "circle", utils.Vec2f{}))
			if moving {
				v, _ := x.Velocities.Get(circle)
				v.SetLocalVector(utils.Vec2f{Y: 1})
			}

			cols := physicsCollisions(x, []common.EntityPair{{A: rect, B: circle}})
			if len(cols) != 1 || !testutil.ApproxEqual(cols[0].Vector, c.want) {
				t.Errorf("circle at %v inside a 10×10 rectangle, moving %v: got %+v, want push-out %v", c.offset, moving, cols, c.want)
			}
		}
	}
}
