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

func looseWood(t *testing.T, s *testutil.Session, tile utils.Vec2i, count int) common.EntityId {
	t.Helper()
	wood, err := s.ECS.SpawnItemStack(data.Item_Wood, count, utils.TilePosToWorldPos(tile).Add(utils.Vec2f{X: 3, Y: -2}))
	if err != nil {
		t.Fatal(err)
	}
	return wood
}

func TestAColonistGathersWoodLyingOnTheHarvestAreaButNotElsewhere(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	a := groundAway(t, s, colonist, 5)
	b := groundAway(t, s, colonist, 8, a)
	off := groundAway(t, s, colonist, 6, a, b)
	harvest(s, a, b)
	gathered := []common.EntityId{looseWood(t, s, a, 3), looseWood(t, s, a, 1), looseWood(t, s, b, 2)}
	left := looseWood(t, s, off, 5)

	before := woodCarried(s, colonist)
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) == before+6 }) {
		t.Fatalf("the colonist carries %d wood, want the 6 on the harvest area added to its %d", woodCarried(s, colonist), before)
	}
	for _, wood := range gathered {
		if s.ECS.Parent(wood) == -1 && s.ECS.HasEntity(wood) {
			t.Fatalf("wood %d is still lying on the harvest area", wood)
		}
	}
	live(t, s, 5*data.TPS, func() bool { return false })
	if s.ECS.Parent(left) != -1 || colonistState(s, colonist) != inputsources.Colonist_Idle {
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

func TestAnItemIsOnlyPickedUpOnPurposeFromCloseBy(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	pos := testutil.WorldPos(t, s.ECS, colonist)
	far, err := s.ECS.SpawnItemStack(data.Item_Wood, 1, pos.Add(utils.Vec2f{X: data.ItemPickUpOnPurposeReachPx + 1}))
	if err != nil {
		t.Fatal(err)
	}
	near, err := s.ECS.SpawnItemStack(data.Item_Wood, 1, pos.Add(utils.Vec2f{Y: data.ItemPickUpOnPurposeReachPx}))
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range []common.EntityId{far, near} {
		inputs := map[common.EntityId]ecs.InputState{colonist: {PickUp: ecs.ItemPickUp{PickUp: true, Item: item}}}
		if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
			t.Fatal(err)
		}
		testutil.RemoveScheduled(t, s.ECS)
	}
	if s.ECS.Parent(far) != -1 || !s.ECS.HasEntity(far) {
		t.Fatal("an item out of reach was picked up")
	}
	if s.ECS.HasEntity(near) && s.ECS.Parent(near) == -1 {
		t.Fatal("an item in reach wasn't picked up")
	}
}

func TestWhatAColonistIsMadeToPickUpStaysOffItsHotbar(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	torch, err := s.ECS.SpawnItemStack(data.Item_Torch, 1, testutil.WorldPos(t, s.ECS, colonist))
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := s.ECS.PickUpItem(colonist, torch); err != nil || !ok {
		t.Fatalf("the colonist couldn't be made to pick up a torch: %v", err)
	}
	inv, _ := s.ECS.Inventories.Get(colonist)
	if s.ECS.Parent(torch) != colonist || inv.HotbarSlotOf(torch) != -1 {
		t.Fatal("the torch didn't go in a slot of its own, off the colonist's hotbar")
	}
}

func TestAColonistOnlyGathersWhatsHarvested(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	lying := groundAway(t, s, colonist, 5)
	harvest(s, lying)
	torch, err := s.ECS.SpawnItemStack(data.Item_Torch, 1, utils.TilePosToWorldPos(lying))
	if err != nil {
		t.Fatal(err)
	}

	live(t, s, 5*data.TPS, func() bool { return false })
	if s.ECS.Parent(torch) != -1 || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatalf("the colonist went for a torch lying on the harvest area, it's %v", colonistState(s, colonist))
	}
}
