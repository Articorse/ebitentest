package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/abilitysystem"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

func addSwingingTool(t *testing.T, x *ecs.ECSContainer, cooldownMs int, swing *ecs.Swing) (tool, holder common.EntityId) {
	t.Helper()
	const abilityId = ecs.AbilityEnum(1004)
	if err := x.RegisterAbility(abilityId, func(common.EntityId, ecs.AbilityTarget, ecs.AbilityParams, *ecs.ECSContainer) (bool, error) {
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}

	abis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{{Def: ecs.NewAbilityDef(abilityId, ecs.Ability_None, cooldownMs, 0)}}
	tool = testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewEquipmentComponent(ecs.Equipable_MainHand, abis), swing)
	holder = testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewAimComponent(),
		ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{ecs.Equip_MainHand: tool}))
	testutil.Attach(t, x, tool, holder, ecs.Link_Held)
	return tool, holder
}

func TestSwingTurnsQuicklyThenComesBackOverTheCooldown(t *testing.T) {
	x := testutil.NewECS(t)
	const angle = 2.0
	tool, holder := addSwingingTool(t, x, 20*data.TickMs, ecs.NewSwingComponent(angle, 0.25, utils.Vec2f{}))

	if a := x.SwingAngle(tool); a != 0 {
		t.Fatalf("angle %v before use, want 0", a)
	}
	if ok, err := x.ActivateEquipmentAbility(holder, ecs.Equip_MainHand, ecs.AbilityTarget{}, 0); err != nil || !ok {
		t.Fatalf("activated %v, error %v", ok, err)
	}

	// The first tick ends the ability's zero-length activity and starts its cooldown
	want := map[int]float64{1: 0, 3: 0.4 * angle, 6: angle, 11: angle * 2 / 3, 16: angle / 3, 21: 0, 25: 0}
	for tick := 1; tick <= 25; tick++ {
		if err := abilitysystem.Tick(x); err != nil {
			t.Fatal(err)
		}
		if w, ok := want[tick]; ok {
			if a := x.SwingAngle(tool); math.Abs(a-w) > 1e-9 {
				t.Errorf("angle %v after %d ticks, want %v", a, tick, w)
			}
		}
	}
}

func TestSwingMirrorsWhenAimingLeftOfTheHolder(t *testing.T) {
	x := testutil.NewECS(t)
	tool, holder := addSwingingTool(t, x, 100, ecs.NewSwingComponent(1, 0.5, utils.Vec2f{}))
	testutil.SetPos(t, x, holder, utils.Vec2f{X: 50, Y: 50})
	sw, _ := x.Swings.Get(tool)

	for _, c := range []struct {
		aim    utils.Vec2f
		hasAim bool
		want   bool
	}{
		{utils.Vec2f{X: 40, Y: 90}, true, true},
		{utils.Vec2f{}, false, true},
		{utils.Vec2f{X: 60, Y: 10}, true, false},
		{utils.Vec2f{}, false, false},
		{utils.Vec2f{}, true, true},
	} {
		input := ecs.InputState{AimPos: c.aim, HasAim: c.hasAim}
		if err := inputsystem.HandleInputs(utils.Vec2f{}, x, map[common.EntityId]ecs.InputState{holder: input}); err != nil {
			t.Fatal(err)
		}
		if sw.Mirrored() != c.want {
			t.Fatalf("aiming at %v: mirrored %v, want %v", c.aim, sw.Mirrored(), c.want)
		}
	}
}

func TestSwingIsSaved(t *testing.T) {
	x := testutil.NewECS(t)
	tool, _ := addSwingingTool(t, x, 100, ecs.NewSwingComponent(1.5, 0.2, utils.Vec2f{X: 3, Y: 4}))
	sw, _ := x.Swings.Get(tool)
	sw.SetMirrored(true)

	members, err := x.GetCompositionComponents(tool)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)

	restored, ok := y.Swings.Get(tool)
	if !ok || *restored != *sw {
		t.Fatalf("restored swing %+v (%v), want %+v", restored, ok, sw)
	}
}

func TestSwingKeepsItsSideUntilItsReadyAgain(t *testing.T) {
	x := testutil.NewECS(t)
	tool, holder := addSwingingTool(t, x, 10*data.TickMs, ecs.NewSwingComponent(1, 0.5, utils.Vec2f{}))
	sw, _ := x.Swings.Get(tool)
	left, right := utils.Vec2f{X: -10}, utils.Vec2f{X: 10}
	aim := func(pos utils.Vec2f, use bool) {
		t.Helper()
		input := ecs.InputState{AimPos: pos, HasAim: true}
		if use {
			input.MainHandEqAbility1 = 1
		}
		if err := inputsystem.HandleInputs(utils.Vec2f{}, x, map[common.EntityId]ecs.InputState{holder: input}); err != nil {
			t.Fatal(err)
		}
	}

	aim(left, true)
	if !sw.Mirrored() || !x.IsSwinging(tool) {
		t.Fatalf("used aiming left: mirrored %v, swinging %v, want both", sw.Mirrored(), x.IsSwinging(tool))
	}

	for range 11 {
		aim(right, false)
		if !sw.Mirrored() {
			t.Fatal("the swing flipped sides before it was ready again")
		}
		if err := abilitysystem.Tick(x); err != nil {
			t.Fatal(err)
		}
	}

	aim(right, false)
	if x.IsSwinging(tool) || sw.Mirrored() {
		t.Fatalf("once ready: swinging %v, mirrored %v, want neither", x.IsSwinging(tool), sw.Mirrored())
	}
}
