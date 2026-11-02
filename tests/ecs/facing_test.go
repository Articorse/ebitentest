package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/facingsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

const deg = math.Pi / 180

func approxAngle(a, b float64) bool {
	return math.Abs(utils.NormalizeAngle(a-b)) < 1e-9
}

func TestFacingTurns(t *testing.T) {
	perTick := 90 * deg * data.TickMs / 1000
	cases := []struct {
		name            string
		facing          *ecs.Facing
		current, target float64
		want            float64
	}{
		{"instantly", ecs.NewFacingComponent(), 10 * deg, 170 * deg, 170 * deg},
		{"instantly the short way round", ecs.NewFacingComponent(), 170 * deg, -170 * deg, -170 * deg},
		{"at its turn rate", ecs.NewFacingComponent().WithTurnRate(90 * deg), 0, 90 * deg, perTick},
		{"at its turn rate, the short way round", ecs.NewFacingComponent().WithTurnRate(90 * deg), 170 * deg, -170 * deg, 170*deg + perTick},
		{"without overshooting", ecs.NewFacingComponent().WithTurnRate(90 * deg), 0, perTick / 2, perTick / 2},
		{"to the edge of its arc", ecs.NewFacingComponent().WithAngleLimits(0, 90*deg), 0, 80 * deg, 45 * deg},
		{"to the edge of an arc off center", ecs.NewFacingComponent().WithAngleLimits(90*deg, 90*deg), 90 * deg, 0, 45 * deg},
		{"back into its arc", ecs.NewFacingComponent().WithAngleLimits(0, 90*deg), 120 * deg, 10 * deg, 10 * deg},
		{
			"the long way round inside its arc",
			ecs.NewFacingComponent().WithTurnRate(90*deg).WithAngleLimits(0, 270*deg),
			120 * deg, -120 * deg, 120*deg - perTick,
		},
	}

	for _, c := range cases {
		if got := c.facing.Turn(c.current, c.target); !approxAngle(got, c.want) {
			t.Errorf("%s: turned to %v°, want %v°", c.name, got/deg, c.want/deg)
		}
	}
}

func addFacing(t *testing.T, x *ecs.ECSContainer, parent common.EntityId, pos utils.Vec2f, facing *ecs.Facing) common.EntityId {
	t.Helper()
	e := testutil.AddMovable(t, x, pos, false, facing)
	if parent != -1 {
		testutil.Attach(t, x, e, parent, ecs.Link_Held)
	}
	return e
}

func worldRotation(t *testing.T, x *ecs.ECSContainer, e common.EntityId) float64 {
	t.Helper()
	r, err := x.WorldRotation(e)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestFacingPointsFromItselfAtItsHoldersAim(t *testing.T) {
	x := testutil.NewECS(t)
	holder := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewAimComponent())
	holderTransform, _ := x.Transforms.Get(holder)
	holderTransform.SetLocalRotation(90 * deg)
	item := addFacing(t, x, holder, utils.Vec2f{X: 10}, ecs.NewFacingComponent())
	aim, _ := x.Aims.Get(holder)
	aim.SetTarget(utils.Vec2f{X: 10, Y: -20})

	facingsystem.Tick(x)

	if r := worldRotation(t, x, item); !approxAngle(r, -90*deg) {
		t.Fatalf("faces %v°, want -90°: straight up from the item, not from its holder, whatever the holder's rotation", r/deg)
	}
}

func TestFacingLimitsAreAroundTheParentsRotation(t *testing.T) {
	x := testutil.NewECS(t)
	holder := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewAimComponent())
	holderTransform, _ := x.Transforms.Get(holder)
	holderTransform.SetLocalRotation(90 * deg)
	item := addFacing(t, x, holder, utils.Vec2f{}, ecs.NewFacingComponent().WithAngleLimits(0, 90*deg))
	aim, _ := x.Aims.Get(holder)
	aim.SetTarget(utils.Vec2f{X: 10})

	facingsystem.Tick(x)

	if r := worldRotation(t, x, item); !approxAngle(r, 45*deg) {
		t.Fatalf("faces %v°, want 45°: as far right as 45° either side of its holder's 90° allows", r/deg)
	}
}

func TestFacingWithoutAnAimOrWhileDisabledStaysPut(t *testing.T) {
	x := testutil.NewECS(t)
	holder := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewAimComponent())
	item := addFacing(t, x, holder, utils.Vec2f{}, ecs.NewFacingComponent())
	aim, _ := x.Aims.Get(holder)

	aim.SetTarget(utils.Vec2f{Y: 10})
	facingsystem.Tick(x)
	aim.ClearTarget()
	facingsystem.Tick(x)
	if r := worldRotation(t, x, item); !approxAngle(r, 90*deg) {
		t.Fatalf("faces %v° after its aim was cleared, want the 90° it last aimed at", r/deg)
	}

	aim.SetTarget(utils.Vec2f{X: -10})
	x.SetDisabled(item, true)
	facingsystem.Tick(x)
	if r := worldRotation(t, x, item); !approxAngle(r, 90*deg) {
		t.Fatalf("faces %v° after turning while disabled, want 90°", r/deg)
	}
}

func TestAimOfIsTheClosestAimUpTheHierarchy(t *testing.T) {
	x := testutil.NewECS(t)
	holder := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewAimComponent())
	item := addFacing(t, x, holder, utils.Vec2f{}, ecs.NewFacingComponent())
	scope := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewAimComponent())
	testutil.Attach(t, x, scope, item, ecs.Link_Attached)
	loose := testutil.AddMovable(t, x, utils.Vec2f{}, false)

	holderAim, _ := x.Aims.Get(holder)
	holderAim.SetTarget(utils.Vec2f{X: 1})
	scopeAim, _ := x.Aims.Get(scope)
	scopeAim.SetTarget(utils.Vec2f{X: 2})

	for _, c := range []struct {
		e    common.EntityId
		want utils.Vec2f
		ok   bool
	}{
		{holder, utils.Vec2f{X: 1}, true},
		{item, utils.Vec2f{X: 1}, true},
		{scope, utils.Vec2f{X: 2}, true},
		{loose, utils.Vec2f{}, false},
	} {
		if got, ok := x.AimOf(c.e); got != c.want || ok != c.ok {
			t.Errorf("aim of %d is %v (%v), want %v (%v)", c.e, got, ok, c.want, c.ok)
		}
	}
}

func TestAimAndFacingAreSaved(t *testing.T) {
	x := testutil.NewECS(t)
	facing := ecs.NewFacingComponent().WithTurnRate(3).WithAngleLimits(1, 2)
	e := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewAimComponent(), facing)
	aim, _ := x.Aims.Get(e)
	aim.SetTarget(utils.Vec2f{X: 4, Y: 5})

	members, err := x.GetCompositionComponents(e)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)

	restoredAim, _ := y.Aims.Get(e)
	restoredFacing, _ := y.Facings.Get(e)
	if restoredAim == nil || *restoredAim != *aim || restoredFacing == nil || *restoredFacing != *facing {
		t.Fatalf("restored aim %+v and facing %+v, want %+v and %+v", restoredAim, restoredFacing, aim, facing)
	}
}
