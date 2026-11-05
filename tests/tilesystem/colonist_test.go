package tilesystem_test

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/itemsystem"
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
	if !ok || inv.Capacity() != 9 || inv.HotbarSize() != 9 || !inv.ManualPickup() {
		t.Fatal("the colonist doesn't have 9 slots, all on the hotbar, that only pick up what they're told to")
	}
	wood, _ := inv.Slot(0)
	if item, ok := s.ECS.Items.Get(wood); !ok || item.Kind() != data.Item_Wood || item.Count() != 20 {
		t.Fatal("the colonist doesn't start with 20 wood")
	}
	if _, inHand := inv.HotbarItem(inv.Selected()); inHand || inv.HotbarSlotOf(wood) != -1 {
		t.Fatal("the colonist's wood is on its hotbar, which it only uses for what it's using")
	}
}

func TestAColonistOnlyPicksUpWhatItsToldTo(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	_, colonist := spawnColonist(t, s)
	wood, err := s.ECS.SpawnItem(data.Item_Wood, testutil.WorldPos(t, s.ECS, colonist))
	if err != nil {
		t.Fatal(err)
	}

	for range 10 {
		itemsystem.Tick(s.ECS)
	}
	if s.ECS.Parent(wood) != -1 {
		t.Fatal("the colonist picked up wood lying under it")
	}
	if ok, err := s.ECS.PickUpItem(colonist, wood); err != nil || !ok {
		t.Fatal("the colonist couldn't be made to pick up the wood")
	}
	testutil.RemoveScheduled(t, s.ECS)
	inv, _ := s.ECS.Inventories.Get(colonist)
	stack, _ := inv.Slot(0)
	if item, _ := s.ECS.Items.Get(stack); item.Count() != 21 {
		t.Fatalf("the colonist's stack has %d wood, want the 21 with the one picked up", item.Count())
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
	if !ok || !inv.ManualPickup() {
		t.Fatal("after restarting, the colonist is gone or picks up whatever it comes across")
	}
	if wood, _ := inv.Slot(0); !r.ECS.Items.HasComponent(wood) {
		t.Fatal("after restarting, the colonist lost its wood")
	}
}
