package content_test

import (
	"ebittest/content/inputsources"
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/ecs/systems/damagesystem"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/ecs/systems/visionsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

func playerAndSlime(t *testing.T, slimePos utils.Vec2f) (*ecs.ECSContainer, common.EntityId, common.EntityId) {
	t.Helper()
	x := testutil.NewECS(t)
	player, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	slime, err := templates.SpawnEvilSlime(x, slimePos, player)
	if err != nil {
		t.Fatal(err)
	}
	return x, player, slime
}

func bodyHealth(t *testing.T, x *ecs.ECSContainer, e common.EntityId) int {
	t.Helper()
	parts, _, err := x.BodyStatus(e)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, p := range parts {
		total += p.Health
	}
	return total
}

func TestTheEvilSlimeHasThePlayersBody(t *testing.T) {
	x, player, slime := playerAndSlime(t, utils.Vec2f{X: 100})
	playerBody, _ := x.Bodies.Get(player)
	slimeBody, _ := x.Bodies.Get(slime)
	if slimeBody.Kind() != playerBody.Kind() {
		t.Fatalf("the slime's body is %v, want the player's %v", slimeBody.Kind(), playerBody.Kind())
	}
	if err := x.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestTheEvilSlimeFollowsThePlayerAtHalfItsSpeed(t *testing.T) {
	x, player, slime := playerAndSlime(t, utils.Vec2f{X: 30, Y: 40})
	x.SetTerrain(newFakeTerrain())
	visionsystem.Tick(x)
	in, _ := x.Inputs.Get(slime)
	follow, err := x.InputSourceFunc(in.Type())
	if err != nil {
		t.Fatal(err)
	}
	toPlayer, err := follow(slime, 0, in.Params(), x)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(toPlayer.Analog1X+0.6) > 1e-9 || math.Abs(toPlayer.Analog1Y+0.8) > 1e-9 {
		t.Fatalf("the slime at (30, 40) heads %v, %v, want straight at the player at the origin", toPlayer.Analog1X, toPlayer.Analog1Y)
	}

	if err := inputsystem.HandleInputs(utils.Vec2f{}, x, map[common.EntityId]ecs.InputState{
		slime:  toPlayer,
		player: {Analog1X: toPlayer.Analog1X, Analog1Y: toPlayer.Analog1Y},
	}); err != nil {
		t.Fatal(err)
	}
	slimeVel, _ := x.Velocities.Get(slime)
	playerVel, _ := x.Velocities.Get(player)
	if got, want := slimeVel.LocalVector().Length(), playerVel.LocalVector().Length()/2; math.Abs(got-want) > 1e-9 || want == 0 {
		t.Fatalf("the slime sped up by %v, want half the player's %v", got, want*2)
	}
}

func TestTheEvilSlimeHurtsThePlayerFor5EverySecondOfContact(t *testing.T) {
	x, player, slime := playerAndSlime(t, utils.Vec2f{X: 4})
	full := bodyHealth(t, x, player)
	hurtboxes, hitboxes := ecs.Colliders(&x.HurtboxColliders), ecs.Colliders(&x.HitboxColliders)
	touch := func() {
		t.Helper()
		pairs := collisionsystem.GetDirectedAABBCollisions(hurtboxes, hitboxes, []common.EntityPair{{A: slime, B: player}, {A: player, B: slime}}, x)
		if _, err := damagesystem.DealContactDamage(collisionsystem.GetCollisions(hurtboxes, hitboxes, pairs, x), nil, x); err != nil {
			t.Fatal(err)
		}
		damagesystem.Tick(x)
	}

	touch()
	if lost := full - bodyHealth(t, x, player); lost != 5 {
		t.Fatalf("touching the slime cost the player %d health, want 5", lost)
	}
	if vel, _ := x.Velocities.Get(player); vel.LocalVector().X >= 0 {
		t.Fatalf("the player was knocked %v, want away from the slime on its right", vel.LocalVector())
	}
	if bodyHealth(t, x, slime) != full {
		t.Fatal("the player's touch hurt the slime")
	}

	for range 1000/data.TickMs - 2 {
		touch()
	}
	if lost := full - bodyHealth(t, x, player); lost != 5 {
		t.Fatalf("within a second of touching, the player lost %d health, want 5", lost)
	}
	for range 3 {
		touch()
	}
	if lost := full - bodyHealth(t, x, player); lost != 10 {
		t.Fatalf("after a second of touching, the player lost %d health, want 10", lost)
	}
}

func rush(t *testing.T, x *ecs.ECSContainer, slime common.EntityId) (ecs.InputState, inputsources.RusherState) {
	t.Helper()
	visionsystem.Tick(x)
	in, _ := x.Inputs.Get(slime)
	rusher, err := x.InputSourceFunc(in.Type())
	if err != nil {
		t.Fatal(err)
	}
	input, err := rusher(slime, 0, in.Params(), x)
	if err != nil {
		t.Fatal(err)
	}
	return input, in.Params().(*inputsources.InputRusherParams).State
}

func TestTheEvilSlimeIdlesUntilItSeesThePlayerAndRushesWhileItDoes(t *testing.T) {
	// Lit within 5 tiles of the origin, and dark beyond
	x, player, slime := playerAndSlime(t, utils.TilePosToWorldPos(utils.Vec2i{X: 3}))
	x.SetTerrain(newFakeTerrain())
	if v, ok := x.Visions.Get(slime); !ok || v.BaseRangeTiles() != data.PlayerVisionRangeTiles/2 {
		t.Fatal("the slime doesn't see half as far as the player")
	}

	if err := x.SetWorldPos(player, utils.TilePosToWorldPos(utils.Vec2i{X: 20})); err != nil {
		t.Fatal(err)
	}
	if input, state := rush(t, x, slime); state != inputsources.Rusher_Idle || input != (ecs.InputState{}) {
		t.Fatalf("with the player in the dark, the slime is in state %v moving %v, %v; want it idle", state, input.Analog1X, input.Analog1Y)
	}

	if err := x.SetWorldPos(player, utils.TilePosToWorldPos(utils.Vec2i{X: -2})); err != nil {
		t.Fatal(err)
	}
	if input, state := rush(t, x, slime); state != inputsources.Rusher_Follow || input.Analog1X != -1 || input.Analog1Y != 0 {
		t.Fatalf("with the player in sight to its left, the slime is in state %v moving %v, %v; want it rushing left", state, input.Analog1X, input.Analog1Y)
	}

	x.SetTerrain(newFakeTerrain(utils.Vec2i{X: 0}, utils.Vec2i{X: 0, Y: 1}, utils.Vec2i{X: 0, Y: -1}))
	if input, state := rush(t, x, slime); state != inputsources.Rusher_Idle || input != (ecs.InputState{}) {
		t.Fatalf("with the player behind a wall, the slime is in state %v moving %v, %v; want it idle again", state, input.Analog1X, input.Analog1Y)
	}

	x.SetTerrain(newFakeTerrain())
	x.ScheduleKillEntity(player)
	testutil.RemoveScheduled(t, x)
	if _, state := rush(t, x, slime); state != inputsources.Rusher_Idle {
		t.Fatalf("with the player gone, the slime is in state %v, want it idle", state)
	}
}

func TestTheRushersStateIsSaved(t *testing.T) {
	p := &inputsources.InputRusherParams{Target: 3, State: inputsources.Rusher_Follow}
	c := p.Copy().(*inputsources.InputRusherParams)
	if c.Target != p.Target || c.State != p.State {
		t.Fatalf("the copy is %+v, want %+v", *c, *p)
	}
	c.State = inputsources.Rusher_Idle
	if p.State != inputsources.Rusher_Follow {
		t.Fatal("changing the copy changed the original")
	}
}
