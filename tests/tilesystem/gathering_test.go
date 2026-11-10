package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func looseWood(t *testing.T, s *testutil.Session, tile utils.Vec2i, mg int) {
	t.Helper()
	if laid := s.ECS.LayDownKind(data.Item_Wood, mg, tile); laid != mg {
		t.Fatalf("laid down %d of %d mg of wood on %v", laid, mg, tile)
	}
	if kind, _ := onGround(s, tile); kind != data.Item_Wood {
		t.Fatalf("the wood didn't go on %v", tile)
	}
}

// So that what it carries doesn't keep it from picking up much
func withoutWood(t *testing.T, s *testutil.Session, colonist common.EntityId) {
	t.Helper()
	for woodCarried(s, colonist) > 0 {
		inv, _ := s.ECS.Inventories.Get(colonist)
		wood, _ := inv.Slot(0)
		comp, _ := s.ECS.Items.Get(wood)
		if err := s.ECS.UseUp(wood, comp.Amount()); err != nil {
			t.Fatal(err)
		}
	}
	testutil.RemoveScheduled(t, s.ECS)
}

func TestAColonistGathersWoodLyingOnTheHarvestAreaButNotElsewhere(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	a := groundAway(t, s, colonist, 5)
	b := groundAway(t, s, colonist, 8, a)
	off := groundAway(t, s, colonist, 6, a, b)
	harvest(s, a, b)
	looseWood(t, s, a, 4*kg)
	looseWood(t, s, b, 2*kg)
	looseWood(t, s, off, 5*kg)
	withoutWood(t, s, colonist)

	before := woodCarried(s, colonist)
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) == before+6*kg }) {
		t.Fatalf("the colonist carries %d wood, want the 6 on the harvest area added to its %d", woodCarried(s, colonist), before)
	}
	for _, tile := range []utils.Vec2i{a, b} {
		if _, ok := s.Chunks.GroundStack(tile); ok {
			t.Fatalf("wood is still lying on %v on the harvest area", tile)
		}
	}
	live(t, s, 5*data.TPS, func() bool { return false })
	if _, n := onGround(s, off); n != 5*kg || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatalf("the colonist went for wood off the harvest area, it's %v", colonistState(s, colonist))
	}

	inv, _ := s.ECS.Inventories.Get(colonist)
	for i := range inv.HotbarSize() {
		if item, bound := inv.HotbarItem(i); bound {
			t.Fatalf("what the colonist gathered went on its hotbar: %d on slot %d", item, i+1)
		}
	}
}

func TestGatheringComesAfterDeliveringAndBeforeFellingAsFarAway(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 8)
	lying := groundAway(t, s, colonist, 8, tree)
	harvest(s, tree, lying)
	looseWood(t, s, lying, 4*kg)
	site := groundAway(t, s, colonist, 8, tree, lying)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)

	live(t, s, 1, func() bool { return false })
	if state := colonistState(s, colonist); state != inputsources.Colonist_Deliver {
		t.Fatalf("the colonist is %v first, want delivering", state)
	}
	gathered := func() bool { _, ok := s.Chunks.GroundStack(lying); return !ok }
	if !live(t, s, 30*data.TPS, gathered) {
		t.Fatalf("the colonist didn't gather the wood, it's %v", colonistState(s, colonist))
	}
	if !isTree(s, tree) {
		t.Fatal("the colonist felled the tree before gathering the wood")
	}
}

func TestWhatLiesOnTheGroundIsOnlyPickedUpFromWithinReachAndSight(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := storageSpot(t, s, colonist, 6, data.ItemReachTiles+2, 1)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(row))
	near := utils.Vec2i{X: row.X + data.ItemReachTiles, Y: row.Y}
	far := utils.Vec2i{X: row.X + data.ItemReachTiles + 1, Y: row.Y}
	looseWood(t, s, near, 1*kg)
	looseWood(t, s, far, 1*kg)
	before := woodCarried(s, colonist)

	for _, tile := range []utils.Vec2i{far, near} {
		inputs := map[common.EntityId]ecs.InputState{colonist: {PickUp: ecs.ItemPickUp{PickUp: true, Tile: tile}}}
		if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
			t.Fatal(err)
		}
		testutil.RemoveScheduled(t, s.ECS)
	}
	if _, ok := s.Chunks.GroundStack(far); !ok {
		t.Fatal("wood out of reach was picked up")
	}
	if _, ok := s.Chunks.GroundStack(near); ok || woodCarried(s, colonist) != before+1*kg {
		t.Fatal("wood in reach wasn't picked up")
	}
}

func TestWhatAColonistPicksUpStaysOffItsHotbar(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 1)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(row))
	s.ECS.LayDownKind(data.Item_Torch, 1, row)
	if n, err := s.ECS.PickUpFromGround(colonist, row); err != nil || n != 1 {
		t.Fatalf("the colonist couldn't pick up a torch: %v", err)
	}
	inv, _ := s.ECS.Inventories.Get(colonist)
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok {
			if comp, _ := s.ECS.Items.Get(item); comp.Kind() == data.Item_Torch && inv.HotbarSlotOf(item) != -1 {
				t.Fatal("the torch went on the colonist's hotbar")
			}
		}
	}
}

func TestAColonistOnlyGathersWhatsHarvested(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	lying := groundAway(t, s, colonist, 5)
	harvest(s, lying)
	s.ECS.LayDownKind(data.Item_Torch, 1, lying)

	live(t, s, 5*data.TPS, func() bool { return false })
	if kind, _ := onGround(s, lying); kind != data.Item_Torch || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatalf("the colonist went for a torch lying on the harvest area, it's %v", colonistState(s, colonist))
	}
}
