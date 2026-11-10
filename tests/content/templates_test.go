package content_test

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestPlayerTemplateWithStartingEquipment(t *testing.T) {
	x := testutil.NewECS(t)
	p, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	if err := templates.SpawnStartingEquipment(x, p); err != nil {
		t.Fatal(err)
	}

	if err := x.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
	if !x.ChunkLoaders.HasComponent(p) {
		t.Fatal("player isn't a chunkloader")
	}

	members, err := x.GetCompositionComponents(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 7 {
		t.Fatalf("player stored with %d entities, want itself, the mattock, the bazooka, the gun, the axe, the torches and the spacesuit", len(members))
	}

	if body, ok := x.Bodies.Get(p); !ok || body.Kind() != data.Body_Humanoid {
		t.Fatal("the player doesn't have a humanoid body")
	}
	if inv, ok := x.Invulnerabilities.Get(p); !ok || inv.PostHitMs() != 1000 {
		t.Fatal("the player isn't invulnerable for a second after each hit")
	}

	inv, _ := x.Inventories.Get(p)
	torches, _ := inv.Slot(4)
	if item, ok := x.Items.Get(torches); !ok || item.Kind() != data.Item_Torch || item.Amount() != 100 {
		t.Fatal("the player's fifth slot doesn't hold 100 torches")
	}
}

func TestStartingEquipmentIsHeldWherePlayerIs(t *testing.T) {
	x := testutil.NewECS(t)
	p, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	spawn := utils.Vec2f{X: 48, Y: -32}
	testutil.SetPos(t, x, p, spawn)
	if err := templates.SpawnStartingEquipment(x, p); err != nil {
		t.Fatal(err)
	}

	equipper, _ := x.Equippers.Get(p)
	for _, e := range equipper.Equipment() {
		if got := testutil.WorldPos(t, x, e); !testutil.ApproxEqual(got, spawn) {
			t.Fatalf("equipment %d is at %v, want the player's position %v", e, got, spawn)
		}
	}
}

func TestPlayerStartsWithItsToolsInOrderAndTheMattockSelected(t *testing.T) {
	x := testutil.NewECS(t)
	p, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	if err := templates.SpawnStartingEquipment(x, p); err != nil {
		t.Fatal(err)
	}

	inv, ok := x.Inventories.Get(p)
	if !ok || inv.Capacity() != 36 || inv.HotbarSize() != data.HotbarSlots {
		t.Fatalf("player inventory %v with %d slots and a hotbar of %d, want 4×9 and %d", ok, inv.Capacity(), inv.HotbarSize(), data.HotbarSlots)
	}
	for i, want := range []common.ImageAssetTag{common.AssetImageMattock, common.AssetSheetBazooka, common.AssetSheetGun, common.AssetImageAxe} {
		item, _ := inv.Slot(i)
		if spr, ok := x.Sprites.Get(item); !ok || spr.ImageAsset() != want {
			t.Fatalf("slot %d holds entity %d, want the %s", i, item, want)
		}
		if bound, _ := inv.HotbarItem(i); bound != item {
			t.Fatalf("hotbar slot %d points at %d, want the %s in slot %d", i, bound, want, i)
		}
	}

	equipper, _ := x.Equippers.Get(p)
	inHand, _ := equipper.InSlot(ecs.Equip_MainHand)
	if mattock, _ := inv.Slot(0); inv.Selected() != 0 || inHand != mattock {
		t.Fatalf("slot %d is selected and %d is in hand, want the mattock %d in the first", inv.Selected(), inHand, mattock)
	}
}
