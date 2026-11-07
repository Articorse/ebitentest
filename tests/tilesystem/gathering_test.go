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

func looseWood(t *testing.T, s *testutil.Session, tile utils.Vec2i, count int) {
	t.Helper()
	if laid := s.ECS.LayDownKind(data.Item_Wood, count, tile); laid != count {
		t.Fatalf("laid down %d of %d wood on %v", laid, count, tile)
	}
	if kind, _ := onGround(s, tile); kind != data.Item_Wood {
		t.Fatalf("the wood didn't go on %v", tile)
	}
}

func TestAColonistGathersWoodLyingOnTheHarvestAreaButNotElsewhere(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	a := groundAway(t, s, colonist, 5)
	b := groundAway(t, s, colonist, 8, a)
	off := groundAway(t, s, colonist, 6, a, b)
	harvest(s, a, b)
	looseWood(t, s, a, 4)
	looseWood(t, s, b, 2)
	looseWood(t, s, off, 5)

	before := woodCarried(s, colonist)
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) == before+6 }) {
		t.Fatalf("the colonist carries %d wood, want the 6 on the harvest area added to its %d", woodCarried(s, colonist), before)
	}
	for _, tile := range []utils.Vec2i{a, b} {
		if _, ok := s.Chunks.GroundStack(tile); ok {
			t.Fatalf("wood is still lying on %v on the harvest area", tile)
		}
	}
	live(t, s, 5*data.TPS, func() bool { return false })
	if _, n := onGround(s, off); n != 5 || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatalf("the colonist went for wood off the harvest area, it's %v", colonistState(s, colonist))
	}

	inv, _ := s.ECS.Inventories.Get(colonist)
	for i := range inv.HotbarSize() {
		if item, bound := inv.HotbarItem(i); bound {
			t.Fatalf("what the colonist gathered went on its hotbar: %d on slot %d", item, i+1)
		}
	}
}

func TestGatheringComesAfterDeliveringAndBeforeFelling(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 3)
	lying := groundAway(t, s, colonist, 9, tree)
	harvest(s, tree, lying)
	looseWood(t, s, lying, 4)
	site := groundAway(t, s, colonist, 14, tree, lying)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)
	before := woodCarried(s, colonist)

	if !live(t, s, 20*data.TPS, func() bool {
		got, _ := s.Chunks.Site(site)
		def, _ := testutil.Blueprints(t).Def(data.Blueprint_Planks)
		return got.Supplied(def)
	}) {
		t.Fatalf("the colonist didn't deliver to the blueprint, it's %v", colonistState(s, colonist))
	}
	if woodCarried(s, colonist) != before-planksCost(t) {
		t.Fatal("the colonist gathered before delivering")
	}
	if !live(t, s, 30*data.TPS, func() bool { return woodCarried(s, colonist) == before-planksCost(t)+4 }) {
		t.Fatalf("the colonist didn't gather the wood, it's %v", colonistState(s, colonist))
	}
	if !isTree(s, tree) {
		t.Fatal("the colonist felled the closer tree before gathering the wood")
	}
}

func TestWhatLiesOnTheGroundIsOnlyPickedUpFromWithinReachAndSight(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := storageSpot(t, s, colonist, 6, data.ItemReachTiles+2, 1)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(row))
	near := utils.Vec2i{X: row.X + data.ItemReachTiles, Y: row.Y}
	far := utils.Vec2i{X: row.X + data.ItemReachTiles + 1, Y: row.Y}
	looseWood(t, s, near, 1)
	looseWood(t, s, far, 1)
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
	if _, ok := s.Chunks.GroundStack(near); ok || woodCarried(s, colonist) != before+1 {
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
