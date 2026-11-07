package tilesystem_test

import (
	"ebittest/content/abilities"
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func spawnColonist(t *testing.T, s *testutil.Session) (colony, colonist common.EntityId) {
	t.Helper()
	colony = spawnColony(t, s)
	colonist, err := templates.SpawnColonist(s.ECS, colony)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
	return colony, colonist
}

func TestAColonistStartsAtTheColonyWithThePlayersBodyAndAStackOfWood(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	colony, colonist := spawnColonist(t, s)

	if testutil.WorldPos(t, s.ECS, colonist) != testutil.WorldPos(t, s.ECS, colony) {
		t.Fatal("the colonist isn't at the colony center")
	}
	if body, ok := s.ECS.Bodies.Get(colonist); !ok || body.Kind() != data.Body_Humanoid {
		t.Fatal("the colonist doesn't have the player's humanoid body")
	}
	if f, ok := s.ECS.Factions.Get(colonist); !ok || f.Faction() != data.Faction_Colony {
		t.Fatal("the colonist isn't of the colony")
	}
	if spr, ok := s.ECS.Sprites.Get(colonist); !ok || spr.ImageAsset() != common.AssetSheetGoldSlime {
		t.Fatal("the colonist isn't a gold slime")
	}
	if tether, ok := s.ECS.Tethers.Get(colonist); !ok || tether.Anchor() != colony {
		t.Fatal("the colonist isn't tethered to the colony")
	}
	if !s.ECS.Constructors.HasComponent(colonist) {
		t.Fatal("the colonist can't build")
	}

	inv, ok := s.ECS.Inventories.Get(colonist)
	if !ok || inv.Capacity() != 9 || inv.HotbarSize() != 9 || !inv.PickupsOffHotbar() {
		t.Fatal("the colonist doesn't have 9 slots, all on the hotbar, that only pick up what they're told to")
	}
	wood, _ := inv.Slot(0)
	if item, ok := s.ECS.Items.Get(wood); !ok || item.Kind() != data.Item_Wood || item.Count() != data.ColonistStartingWood {
		t.Fatalf("the colonist doesn't start with %d wood", data.ColonistStartingWood)
	}
	if inv.MaxWeight() != data.ColonistMaxCarryWeight || s.ECS.CarriedWeight(colonist) > inv.MaxWeight() {
		t.Fatalf("the colonist can carry %d and starts out carrying %d, want a limit of %d it's under", inv.MaxWeight(), s.ECS.CarriedWeight(colonist), data.ColonistMaxCarryWeight)
	}
	if vel, ok := s.ECS.Velocities.Get(colonist); !ok || vel.Acceleration() != data.ColonistAcceleration {
		t.Fatal("the colonist doesn't walk at a colonist's pace")
	}
	if _, inHand := inv.HotbarItem(inv.Selected()); inHand || inv.HotbarSlotOf(wood) != -1 {
		t.Fatal("the colonist's wood is on its hotbar, which it only uses for what it's using")
	}
	axe, _ := inv.Slot(inv.Capacity() - 1)
	if eq, ok := s.ECS.Equipments.Get(axe); !ok || !eq.HasAbility(abilities.Ability_Slash) || inv.HotbarSlotOf(axe) != -1 {
		t.Fatal("the colonist doesn't start with an axe in its last slot, off its hotbar")
	}
}

func TestAColonistThatLeavesTheColonyIsBroughtBack(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	colony, colonist := spawnColonist(t, s)

	testutil.SetPos(t, s.ECS, colonist, utils.Vec2f{X: 20 * data.ChunkSize * data.TileSize})
	s.Tick(5)
	if !s.ECS.HasEntity(colonist) || !s.ECS.InColony(colony, utils.WorldPosToTilePos(testutil.WorldPos(t, s.ECS, colonist))) {
		t.Fatal("the colonist wasn't brought back into the colony")
	}
}

func TestAColonistIsSavedWithTheColony(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	_, colonist := spawnColonist(t, s)
	s.Commit()

	r := testutil.StartSession(t)
	inv, ok := r.ECS.Inventories.Get(colonist)
	if !ok || !inv.PickupsOffHotbar() {
		t.Fatal("after restarting, the colonist is gone or picks up whatever it comes across")
	}
	if wood, _ := inv.Slot(0); !r.ECS.Items.HasComponent(wood) {
		t.Fatal("after restarting, the colonist lost its wood")
	}
}
