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

// Everything the colonist carries goes in the storage area, so that it has to fetch
func emptyIntoStorage(t *testing.T, s *testutil.Session, colonist, storage common.EntityId) {
	t.Helper()
	inv, _ := s.ECS.Inventories.Get(colonist)
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok {
			if comp, ok := s.ECS.Items.Get(item); ok && comp.Kind() == data.Item_Wood {
				if _, err := s.ECS.Store(colonist, i, storage, comp.Count()); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	testutil.RemoveScheduled(t, s.ECS)
	if woodCarried(s, colonist) != 0 {
		t.Fatal("the colonist's wood didn't all go in the storage")
	}
}

func TestAColonistFetchesWhatASiteLacksFromStorageAndBuildsIt(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 8, 1, 1)
	storage := placeStorage(t, s, first, first)
	emptyIntoStorage(t, s, colonist, storage)
	before := stored(s, storage)
	site := groundAway(t, s, colonist, 6, first)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)

	if !live(t, s, 2*data.TPS, func() bool { return colonistState(s, colonist) == inputsources.Colonist_Fetch }) {
		t.Fatalf("the colonist didn't go to fetch, it's %v", colonistState(s, colonist))
	}
	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the colonist didn't build the blueprint, it's %v", colonistState(s, colonist))
	}
	if stored(s, storage) != before-planksCost(t) || woodCarried(s, colonist) != 0 {
		t.Fatalf("the storage holds %d and the colonist %d wood, want just what the blueprint cost taken out of %d",
			stored(s, storage), woodCarried(s, colonist), before)
	}
}

func TestAColonistOnlyFetchesWhenStorageMakesUpAllItLacks(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 8, 1, 1)
	storage := placeStorage(t, s, first, first)
	emptyIntoStorage(t, s, colonist, storage)
	if _, err := s.ECS.TakeFromStorage(colonist, storage, data.Item_Wood, stored(s, storage)-planksCost(t)+1); err != nil {
		t.Fatal(err)
	}
	inv, _ := s.ECS.Inventories.Get(colonist)
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok {
			if comp, _ := s.ECS.Items.Get(item); comp.Kind() == data.Item_Wood {
				s.ECS.ScheduleRemoveEntity(item)
			}
		}
	}
	testutil.RemoveScheduled(t, s.ECS)
	site := groundAway(t, s, colonist, 6, first)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)

	live(t, s, 5*data.TPS, func() bool { return false })
	if got, _ := s.Chunks.Site(site); stored(s, storage) != planksCost(t)-1 || len(got.Delivered) > 0 && got.Delivered[0].Count > 0 {
		t.Fatal("the colonist fetched part of what the blueprint lacks")
	}
	if state := colonistState(s, colonist); state != inputsources.Colonist_Idle {
		t.Fatalf("the colonist is %v with not enough in storage", state)
	}

	storeWood(t, s, storage, 1)
	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the colonist didn't build once there was enough in storage, it's %v", colonistState(s, colonist))
	}
}

func TestAColonistKeepsWhatSitesNeedWhenPuttingAway(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 6, 1, 1)
	storage := placeStorage(t, s, first, first)
	canBeWalledIn := func(tile utils.Vec2i) bool {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				n := utils.Vec2i{X: tile.X + dx, Y: tile.Y + dy}
				if n != tile && !s.Chunks.CanPlaceTile(n) && !s.Chunks.IsOpaqueTile(n) {
					return false
				}
			}
		}
		return true
	}
	walled := groundAwayWhere(t, s, colonist, 12, canBeWalledIn, first)
	s.Chunks.PlaceSite(walled, data.Blueprint_Planks)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if wall := (utils.Vec2i{X: walled.X + dx, Y: walled.Y + dy}); wall != walled {
				s.Chunks.DamageTile(wall, 1000)
				if !s.Chunks.PlaceTile(wall, data.Tile_Planks) && !s.Chunks.IsOpaqueTile(wall) {
					t.Fatalf("couldn't wall in the blueprint at %v", wall)
				}
			}
		}
	}
	before := woodCarried(s, colonist)

	most := 0
	if !live(t, s, 20*data.TPS, func() bool {
		most = max(most, stored(s, storage))
		return stored(s, storage) == before-planksCost(t)
	}) {
		t.Fatalf("the storage holds %d wood, want all but what the walled-in blueprint needs of the colonist's %d", stored(s, storage), before)
	}
	live(t, s, 3*data.TPS, func() bool { most = max(most, stored(s, storage)); return false })
	if most > before-planksCost(t) {
		t.Fatalf("the storage held %d wood at one point, more than the colonist could spare", most)
	}
	if woodCarried(s, colonist) != planksCost(t) {
		t.Fatalf("the colonist carries %d wood, want the %d the blueprint needs kept", woodCarried(s, colonist), planksCost(t))
	}
}

func TestStoringAndTakingOnlyMoveWhatsAskedForAndOnlyFromCloseBy(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := storageSpot(t, s, colonist, 6, data.StoreRangeTiles+2, 1)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(row))
	near := placeStorage(t, s, utils.Vec2i{X: row.X + data.StoreRangeTiles, Y: row.Y}, utils.Vec2i{X: row.X + data.StoreRangeTiles, Y: row.Y})
	far := placeStorage(t, s, utils.Vec2i{X: row.X + data.StoreRangeTiles + 1, Y: row.Y}, utils.Vec2i{X: row.X + data.StoreRangeTiles + 1, Y: row.Y})
	storeWood(t, s, far, 10)
	before := woodCarried(s, colonist)
	inv, _ := s.ECS.Inventories.Get(colonist)
	slot := -1
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok && countOf(s, item) > 1 {
			slot = i
		}
	}

	handle := func(in ecs.InputState) {
		t.Helper()
		if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, map[common.EntityId]ecs.InputState{colonist: in}); err != nil {
			t.Fatal(err)
		}
		testutil.RemoveScheduled(t, s.ECS)
	}
	handle(ecs.InputState{Store: ecs.ItemStoring{Store: true, From: slot, Count: 3, Storage: near}})
	if stored(s, near) != 3 || woodCarried(s, colonist) != before-3 {
		t.Fatalf("storing 3 of %d left %d stored and %d carried", before, stored(s, near), woodCarried(s, colonist))
	}
	handle(ecs.InputState{Take: ecs.ItemTaking{Take: true, Kind: data.Item_Wood, Count: 2, Storage: near}})
	if stored(s, near) != 1 || woodCarried(s, colonist) != before-1 {
		t.Fatalf("taking 2 of 3 left %d stored and %d carried", stored(s, near), woodCarried(s, colonist))
	}
	handle(ecs.InputState{Take: ecs.ItemTaking{Take: true, Kind: data.Item_Wood, Count: 5, Storage: far}})
	handle(ecs.InputState{Store: ecs.ItemStoring{Store: true, From: slot, Count: 5, Storage: far}})
	if stored(s, far) != 10 {
		t.Fatal("storing or taking reached the storage out of reach")
	}
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestTakingASplitStackGivesTheTakerItsOwnStack(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 6, 1, 1)
	storage := placeStorage(t, s, first, first)
	emptyIntoStorage(t, s, colonist, storage)
	inStore, _ := s.ECS.Inventories.Get(storage)
	stack, _ := inStore.Slot(0)

	taken, err := s.ECS.TakeFromStorage(colonist, storage, data.Item_Wood, 7)
	if err != nil || taken != 7 {
		t.Fatalf("took %d wood, want 7: %v", taken, err)
	}
	inv, _ := s.ECS.Inventories.Get(colonist)
	own := common.EntityId(-1)
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok && countOf(s, item) == 7 {
			own = item
		}
	}
	if own == -1 || own == stack || countOf(s, stack) != 13 || s.ECS.Parent(stack) != storage {
		t.Fatal("taking part of a stack didn't leave the rest in storage and give the colonist a stack of its own")
	}
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestTakingOnlyTakesTheKindAskedFor(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 6, 2, 1)
	storage := placeStorage(t, s, first, utils.Vec2i{X: first.X + 1, Y: first.Y})
	torches, err := s.ECS.SpawnItemStack(data.Item_Torch, 4, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ECS.PutInInventory(storage, 0, torches); err != nil {
		t.Fatal(err)
	}
	storeWood(t, s, storage, 10)
	before := woodCarried(s, colonist)

	if taken, err := s.ECS.TakeFromStorage(colonist, storage, data.Item_Wood, 3); err != nil || taken != 3 {
		t.Fatalf("took %d wood, want 3: %v", taken, err)
	}
	if countOf(s, torches) != 4 || s.ECS.Parent(torches) != storage || woodCarried(s, colonist) != before+3 {
		t.Fatal("taking wood took torches, or not the wood")
	}
}
