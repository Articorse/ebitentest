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

func newArmedPlayer(t *testing.T) (x *ecs.ECSContainer, player, mattock, bazooka, gun common.EntityId) {
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
	mattock, _ = inv.Slot(0)
	bazooka, _ = inv.Slot(1)
	gun, _ = inv.Slot(2)
	return x, player, mattock, bazooka, gun
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
	x, player, mattock, bazooka, gun := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	axe, _ := inv.Slot(3)
	torches, _ := inv.Slot(4)

	check := func(wantInHand common.EntityId) {
		t.Helper()
		if got := inHand(x, player); got != wantInHand {
			t.Fatalf("main hand holds %d, want %d", got, wantInHand)
		}
		for _, item := range []common.EntityId{mattock, bazooka, gun, axe, torches} {
			if x.IsDisabled(item) == (item == wantInHand) {
				t.Fatalf("item %d disabled %v, but %d is in hand", item, x.IsDisabled(item), wantInHand)
			}
		}
		if err := x.CheckConsistency(); err != nil {
			t.Fatal(err)
		}
	}

	check(mattock)
	press(t, x, player, ecs.InputState{SelectSlot: 2})
	check(bazooka)
	press(t, x, player, ecs.InputState{SelectSlot: 3})
	check(gun)
	press(t, x, player, ecs.InputState{SelectSlot: 4})
	check(axe)
	press(t, x, player, ecs.InputState{SelectSlot: 5})
	check(torches)
	press(t, x, player, ecs.InputState{SelectSlot: 6})
	check(-1)
	press(t, x, player, ecs.InputState{SelectSlot: 1})
	check(mattock)

	if inv.Selected() != 0 {
		t.Fatalf("slot %d is selected, want the first", inv.Selected())
	}
}

func TestFiringUsesWhatsInHand(t *testing.T) {
	x, player, _, _, gun := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{SelectSlot: 3})

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
	x, player, _, _, _ := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{SelectSlot: 2})
	fire := func() int {
		t.Helper()
		before := len(x.Transforms.GetEntities())
		press(t, x, player, ecs.InputState{MainHandEqAbility1: 1})
		return len(x.Transforms.GetEntities()) - before
	}

	if fire() != 1 {
		t.Fatal("the bazooka didn't fire")
	}
	press(t, x, player, ecs.InputState{SelectSlot: 3})
	for range 500/data.TickMs + 2 {
		if err := abilitysystem.Tick(x); err != nil {
			t.Fatal(err)
		}
	}
	press(t, x, player, ecs.InputState{SelectSlot: 2})

	if fire() != 1 {
		t.Fatal("the bazooka's cooldown didn't run out while it was put away")
	}
}

func TestProjectilesHitTreesAndWalls(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	for _, slot := range []int{2, 3} {
		press(t, x, player, ecs.InputState{SelectSlot: slot})
		before := len(x.Transforms.GetEntities())
		press(t, x, player, ecs.InputState{MainHandEqAbility1: 1})

		spawned := x.Transforms.GetEntities()[before:]
		if len(spawned) == 0 {
			t.Fatalf("slot %d fired nothing", slot)
		}
		for _, e := range spawned {
			hurtbox, ok := x.HurtboxColliders.Get(e)
			if want := ecs.Layer_Wall | ecs.Layer_Feature; !ok || hurtbox.Mask()&want != want {
				t.Fatalf("what slot %d fires doesn't collide with both walls and trees", slot)
			}
		}
	}
}

func TestScrollingStepsThroughTheHotbarAndWraps(t *testing.T) {
	x, player, mattock, bazooka, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)

	press(t, x, player, ecs.InputState{ScrollSlot: 1})
	if inv.Selected() != 1 || inHand(x, player) != bazooka {
		t.Fatalf("scrolling down selected hotbar slot %d, want the bazooka's", inv.Selected()+1)
	}
	press(t, x, player, ecs.InputState{ScrollSlot: -2})
	if inv.Selected() != inv.HotbarSize()-1 {
		t.Fatalf("scrolling up past the first slot selected %d, want the last", inv.Selected()+1)
	}
	press(t, x, player, ecs.InputState{ScrollSlot: 1})
	if inv.Selected() != 0 || inHand(x, player) != mattock {
		t.Fatalf("scrolling down past the last slot selected %d, want the first", inv.Selected()+1)
	}
	press(t, x, player, ecs.InputState{ScrollSlot: 1, SelectSlot: 4})
	if inv.Selected() != 3 {
		t.Fatalf("scrolling while pressing 4 selected %d, want 4", inv.Selected()+1)
	}
}
