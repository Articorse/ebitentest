package systems_test

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/abilitysystem"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func newArmedPlayer(t *testing.T) (x *ecs.ECSContainer, player, bazooka, gun common.EntityId) {
	t.Helper()
	x = testutil.NewECS(t)
	player, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	if err := templates.SpawnStartingEquipment(x, player); err != nil {
		t.Fatal(err)
	}

	inv, _ := x.Inventories.Get(player)
	bazooka, _ = inv.Slot(0)
	gun, _ = inv.Slot(1)
	return x, player, bazooka, gun
}

func inHand(x *ecs.ECSContainer, player common.EntityId) common.EntityId {
	equipper, _ := x.Equippers.Get(player)
	e, ok := equipper.InSlot(ecs.Equip_MainHand)
	if !ok {
		return -1
	}
	return e
}

func press(t *testing.T, x *ecs.ECSContainer, player common.EntityId, input ecs.InputState) {
	t.Helper()
	if err := inputsystem.HandleInputs(utils.Vec2f{}, x, map[common.EntityId]ecs.InputState{player: input}); err != nil {
		t.Fatal(err)
	}
}

func TestOnlyTheSelectedItemIsInHandAndEnabled(t *testing.T) {
	x, player, bazooka, gun := newArmedPlayer(t)

	check := func(wantInHand common.EntityId) {
		t.Helper()
		if got := inHand(x, player); got != wantInHand {
			t.Fatalf("main hand holds %d, want %d", got, wantInHand)
		}
		for _, item := range []common.EntityId{bazooka, gun} {
			if x.IsDisabled(item) == (item == wantInHand) {
				t.Fatalf("item %d disabled %v, but %d is in hand", item, x.IsDisabled(item), wantInHand)
			}
		}
		if err := x.CheckConsistency(); err != nil {
			t.Fatal(err)
		}
	}

	check(bazooka)
	press(t, x, player, ecs.InputState{SelectSlot: 2})
	check(gun)
	press(t, x, player, ecs.InputState{SelectSlot: 3})
	check(-1)
	press(t, x, player, ecs.InputState{SelectSlot: 1})
	check(bazooka)

	inv, _ := x.Inventories.Get(player)
	if inv.Selected() != 0 {
		t.Fatalf("slot %d is selected, want the first", inv.Selected())
	}
}

func TestFiringUsesWhatsInHand(t *testing.T) {
	x, player, _, gun := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{SelectSlot: 2})

	before := len(x.Transforms.GetEntities())
	press(t, x, player, ecs.InputState{MainHandEqAbility1: 1})

	spawned := x.Transforms.GetEntities()[before:]
	if len(spawned) != 1 {
		t.Fatalf("firing spawned %d entities, want one bullet", len(spawned))
	}
	bullet := spawned[0]
	if x.Deathrattles.HasComponent(bullet) || !x.HurtboxColliders.HasComponent(bullet) {
		t.Fatal("firing with the gun in hand didn't spawn a bullet")
	}
	if x.IsDisabled(gun) {
		t.Fatal("the gun was disabled while in hand")
	}
}

func TestCooldownsKeepCountingWhileAnItemIsPutAway(t *testing.T) {
	x, player, _, _ := newArmedPlayer(t)
	fire := func() int {
		t.Helper()
		before := len(x.Transforms.GetEntities())
		press(t, x, player, ecs.InputState{MainHandEqAbility1: 1})
		return len(x.Transforms.GetEntities()) - before
	}

	if fire() != 1 {
		t.Fatal("the bazooka didn't fire")
	}
	press(t, x, player, ecs.InputState{SelectSlot: 2})
	for range 500/data.TickMs + 2 {
		if err := abilitysystem.Tick(x); err != nil {
			t.Fatal(err)
		}
	}
	press(t, x, player, ecs.InputState{SelectSlot: 1})

	if fire() != 1 {
		t.Fatal("the bazooka's cooldown didn't run out while it was put away")
	}
}
