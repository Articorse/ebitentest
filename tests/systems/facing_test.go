package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"math"
	"testing"
)

func TestTheBazookaTurnsSlowlyAndFiresWhereItPoints(t *testing.T) {
	x, player, _, bazooka, _ := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{SelectSlot: 2, AimPos: utils.Vec2f{X: 100}, HasAim: true})

	before := len(x.Transforms.GetEntities())
	press(t, x, player, ecs.InputState{MainHandEqAbility1: 1, AimPos: utils.Vec2f{Y: 100}, HasAim: true})

	oneTick := 2 * math.Pi * data.TickMs / 1000
	if r, _ := x.WorldRotation(bazooka); math.Abs(r-oneTick) > 1e-9 {
		t.Fatalf("the bazooka turned to %v rad in a tick, want %v: a full turn a second", r, oneTick)
	}
	spawned := x.Transforms.GetEntities()[before:]
	if len(spawned) != 1 {
		t.Fatalf("firing spawned %d entities, want one rocket", len(spawned))
	}
	if r, _ := x.WorldRotation(spawned[0]); math.Abs(r-oneTick) > 1e-9 {
		t.Fatalf("the rocket left at %v rad, want where the bazooka points, %v", r, oneTick)
	}
}

func TestTheGunTurnsInstantly(t *testing.T) {
	x, player, _, _, gun := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{SelectSlot: 3, AimPos: utils.Vec2f{X: -100}, HasAim: true})

	if r, _ := x.WorldRotation(gun); math.Abs(math.Abs(r)-math.Pi) > 1e-9 {
		t.Fatalf("the gun points at %v rad, want straight left", r)
	}
}

func TestInputsWithoutAnAimClearIt(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{AimPos: utils.Vec2f{X: 5}, HasAim: true})
	if aim, ok := x.AimOf(player); !ok || aim != (utils.Vec2f{X: 5}) {
		t.Fatalf("aim %v (%v), want {5 0}", aim, ok)
	}

	press(t, x, player, ecs.InputState{AimPos: utils.Vec2f{X: 7}})
	if _, ok := x.AimOf(player); ok {
		t.Fatal("still aiming after an input without an aim")
	}
}
