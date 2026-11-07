package tilesystem_test

import (
	"ebittest/content/abilities"
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

// The top left of a w×h rectangle of open ground about the given distance from the colonist
func storageSpot(t *testing.T, s *testutil.Session, colonist common.EntityId, tiles, w, h int, not ...utils.Vec2i) utils.Vec2i {
	t.Helper()
	return groundAwayWhere(t, s, colonist, tiles, func(tile utils.Vec2i) bool {
		for dy := range h {
			for dx := range w {
				if !s.Chunks.CanPlaceSite(utils.Vec2i{X: tile.X + dx, Y: tile.Y + dy}) {
					return false
				}
			}
		}
		return true
	}, not...)
}

func placeStorage(t *testing.T, s *testutil.Session, first, last utils.Vec2i) common.EntityId {
	t.Helper()
	e, ok, err := s.ECS.PlaceStorage(first, last)
	if err != nil || !ok {
		t.Fatalf("couldn't mark out storage from %v to %v: %v", first, last, err)
	}
	return e
}

func storeWood(t *testing.T, s *testutil.Session, storage common.EntityId, counts ...int) {
	t.Helper()
	pos := testutil.WorldPos(t, s.ECS, storage)
	for _, n := range counts {
		holder, err := s.ECS.AddNewEntity(ecs.NewHierarchyComponent(), ecs.NewTransformComponent(pos, 1, 0), ecs.NewInventoryComponent(1, 0))
		if err != nil {
			t.Fatal(err)
		}
		wood, err := s.ECS.SpawnItemStack(data.Item_Wood, n, pos)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ECS.PutInInventory(holder, 0, wood); err != nil {
			t.Fatal(err)
		}
		if moved, err := s.ECS.Store(holder, 0, storage, n); err != nil || moved != n {
			t.Fatalf("couldn't store %d wood: %v", n, err)
		}
		testutil.RemoveScheduled(t, s.ECS)
	}
}

func stored(s *testutil.Session, storage common.EntityId) int {
	n := 0
	for _, c := range s.ECS.StorageContents(storage) {
		n += c.Count
	}
	return n
}

func TestStorageHasASlotForEachOfItsTilesRowByRow(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 6, 3, 2)
	last := utils.Vec2i{X: first.X + 2, Y: first.Y + 1}
	storage := placeStorage(t, s, first, last)
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}

	inv, _ := s.ECS.Inventories.Get(storage)
	area, _ := s.ECS.StorageAreas.Get(storage)
	if inv.Capacity() != 6 {
		t.Fatalf("a 3×2 storage area has %d slots, want 6", inv.Capacity())
	}
	if area.SlotTile(0) != first || area.SlotTile(2) != (utils.Vec2i{X: first.X + 2, Y: first.Y}) || area.SlotTile(5) != last {
		t.Fatal("the slots aren't the tiles row by row")
	}
	for _, tile := range []utils.Vec2i{first, last} {
		if e, ok := s.ECS.StorageAt(tile); !ok || e != storage {
			t.Fatalf("%v isn't in the storage area", tile)
		}
	}
	if _, ok := s.ECS.StorageAt(utils.Vec2i{X: last.X + 1, Y: last.Y}); ok {
		t.Fatal("the tile past the storage area is in it")
	}
}

func TestStorageOnlyGoesOnOpenGroundInTheColonyAwayFromOtherStorageAndSites(t *testing.T) {
	s, colony, colonist := colonistAtWork(t)
	spot := storageSpot(t, s, colonist, 6, 4, 4)
	placeStorage(t, s, spot, spot)
	cFirst, _, _ := s.ECS.ColonyBounds(colony)
	site := groundAway(t, s, colonist, 12, spot)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)
	tree := treeAway(t, s, colonist, 14, spot, site)
	wall := tileOf(t, s, data.Tile_Rock)

	for _, c := range []struct {
		what        string
		first, last utils.Vec2i
	}{
		{"over other storage", utils.Vec2i{X: spot.X - 1, Y: spot.Y}, utils.Vec2i{X: spot.X + 1, Y: spot.Y + 1}},
		{"over a site", site, site},
		{"on a tree", tree, tree},
		{"on a wall", wall, wall},
		{"across the colony's edge", utils.Vec2i{X: cFirst.X - 1, Y: cFirst.Y}, cFirst},
		{"backwards", utils.Vec2i{X: spot.X + 3, Y: spot.Y + 1}, utils.Vec2i{X: spot.X + 2, Y: spot.Y + 1}},
	} {
		if s.ECS.CanPlaceStorage(c.first, c.last) {
			t.Fatalf("storage can go %s", c.what)
		}
		if _, ok, _ := s.ECS.PlaceStorage(c.first, c.last); ok {
			t.Fatalf("storage was marked out %s", c.what)
		}
	}
	_, cLast, _ := s.ECS.ColonyBounds(colony)
	s.MovePlayer(utils.TilePosToWorldPos(utils.Vec2i{X: cLast.X + 20, Y: 64}))
	outside := utils.Vec2i{}
	if !live(t, s, 200, func() bool {
		for dy := range 40 {
			if outside = (utils.Vec2i{X: cLast.X + 3, Y: 64 + dy}); s.Chunks.WalkSpeedPercent(outside) > 0 {
				return true
			}
		}
		return false
	}) {
		t.Fatal("there's no loaded open ground outside the colony")
	}
	if s.ECS.CanPlaceStorage(outside, outside) {
		t.Fatal("storage can go on open ground outside the colony")
	}
	if !s.ECS.CanPlaceStorage(utils.Vec2i{X: spot.X + 1, Y: spot.Y}, utils.Vec2i{X: spot.X + 3, Y: spot.Y + 3}) {
		t.Fatal("storage can't go right next to other storage")
	}
}

func TestStoringTopsUpWhatsThereThenTakesTheFirstFreeSlot(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 6, 2, 1)
	storage := placeStorage(t, s, first, utils.Vec2i{X: first.X + 1, Y: first.Y})
	storeWood(t, s, storage, 60, 30, 30)

	inv, _ := s.ECS.Inventories.Get(storage)
	counts := []int{}
	for i := range inv.Capacity() {
		item, _ := inv.Slot(i)
		counts = append(counts, countOf(s, item))
	}
	if !slices.Equal(counts, []int{100, 20}) {
		t.Fatalf("storing 60, 30 and 30 wood left %v in the slots, want 100 and 20", counts)
	}
	if contents := s.ECS.StorageContents(storage); len(contents) != 1 || contents[0] != (ecs.ItemCost{Kind: data.Item_Wood, Count: 120}) {
		t.Fatalf("the storage holds %+v, want 120 wood", contents)
	}

	storeWood(t, s, storage, 80)
	if stored(s, storage) != 200 {
		t.Fatalf("the full storage holds %d wood, want 200", stored(s, storage))
	}
	holder, _ := s.ECS.AddNewEntity(ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewInventoryComponent(1, 0))
	wood, _ := s.ECS.SpawnItemStack(data.Item_Wood, 5, utils.Vec2f{})
	if err := s.ECS.PutInInventory(holder, 0, wood); err != nil {
		t.Fatal(err)
	}
	if moved, _ := s.ECS.Store(holder, 0, storage, 5); moved != 0 || s.ECS.HasRoomFor(storage, wood) {
		t.Fatal("wood went into a full storage area")
	}
}

func TestErasingStorageTakesAwayAllItTouchesAndLeavesWhatWasInItOnItsTiles(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	a := storageSpot(t, s, colonist, 6, 3, 1)
	b := storageSpot(t, s, colonist, 12, 1, 1, a)
	erased := placeStorage(t, s, a, utils.Vec2i{X: a.X + 2, Y: a.Y})
	kept := placeStorage(t, s, b, b)
	storeWood(t, s, erased, 100, 7)

	inputs := map[common.EntityId]ecs.InputState{s.Player: {Storage: ecs.StorageMarking{Mark: true, Erase: true, First: utils.Vec2i{X: a.X + 2, Y: a.Y}, Last: utils.Vec2i{X: a.X + 5, Y: a.Y}}}}
	if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
		t.Fatal(err)
	}
	testutil.RemoveScheduled(t, s.ECS)
	if s.ECS.HasEntity(erased) || !s.ECS.HasEntity(kept) {
		t.Fatal("erasing the end of one storage area didn't take away all of it, and just it")
	}
	for i, want := range []int{100, 7} {
		if kind, n := onGround(s, utils.Vec2i{X: a.X + i, Y: a.Y}); kind != data.Item_Wood || n != want {
			t.Fatalf("the tile of slot %d holds %d of %v, want the %d wood that was in it", i+1, n, kind, want)
		}
	}
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestStorageIsMarkedOutByInputAndBlueprintsDontGoOnIt(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 6, 2, 2)
	last := utils.Vec2i{X: first.X + 1, Y: first.Y + 1}
	inputs := map[common.EntityId]ecs.InputState{s.Player: {Storage: ecs.StorageMarking{Mark: true, First: first, Last: last}}}
	if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.ECS.StorageAt(last); !ok {
		t.Fatal("marking out storage didn't")
	}

	inputs[s.Player] = ecs.InputState{PlaceBlueprint: ecs.BlueprintPlacement{Kind: data.Blueprint_Planks, Tiles: []utils.Vec2i{first}}}
	if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Chunks.Site(first); ok {
		t.Fatal("a blueprint went on storage")
	}
}

func TestStorageAndWhatsInItAreSaved(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 6, 2, 1)
	storage := placeStorage(t, s, first, utils.Vec2i{X: first.X + 1, Y: first.Y})
	storeWood(t, s, storage, 100, 3)

	members, err := s.ECS.GetCompositionComponents(storage)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)
	area, ok := y.StorageAreas.Get(storage)
	if !ok {
		t.Fatal("the storage area wasn't saved")
	}
	if f, l := area.Bounds(); f != first || l != (utils.Vec2i{X: first.X + 1, Y: first.Y}) {
		t.Fatalf("the storage area was loaded from %v to %v", f, l)
	}
	if contents := y.StorageContents(storage); len(contents) != 1 || contents[0].Count != 103 {
		t.Fatalf("after loading, the storage holds %+v, want 103 wood", contents)
	}
}

func TestWhatsStoredIsDrawnOnItsTileAndListed(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 6, 3, 1)
	storage := placeStorage(t, s, first, utils.Vec2i{X: first.X + 2, Y: first.Y})
	storeWood(t, s, storage, 100, 5)

	bounds := [2]utils.Vec2f{utils.TilePosToWorldPos(first), utils.TilePosToWorldPos(utils.Vec2i{X: first.X + 2, Y: first.Y})}
	sprites := drawsystem.StoredItemSprites(bounds, s.ECS)
	if len(sprites) != 2 || sprites[0].Pos != utils.TilePosToWorldPos(first) || sprites[1].Pos != utils.TilePosToWorldPos(utils.Vec2i{X: first.X + 1, Y: first.Y}) {
		t.Fatalf("what's stored is drawn as %+v, want on the first two tiles", sprites)
	}
	if lines := drawsystem.StorageTooltip(s.ECS.StorageContents(storage), testutil.ItemAtlas(t)); !slices.Equal(lines, []string{"Storage", "105 Wood"}) {
		t.Fatalf("the storage's tooltip says %q", lines)
	}
	if lines := drawsystem.StorageTooltip(nil, testutil.ItemAtlas(t)); !slices.Equal(lines, []string{"Storage", "Empty"}) {
		t.Fatalf("an empty storage's tooltip says %q", lines)
	}
	if len(drawsystem.StoredItemSprites([2]utils.Vec2f{bounds[1], bounds[1]}, s.ECS)) != 0 {
		t.Fatal("what's stored is drawn off the tiles in view")
	}
}

func TestAColonistPutsAwayWhatItCarriesButItsAxe(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 10, 2, 1)
	storage := placeStorage(t, s, first, utils.Vec2i{X: first.X + 1, Y: first.Y})
	before := woodCarried(s, colonist)

	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) == 0 }) {
		t.Fatalf("the colonist still carries %d wood, it's %v", woodCarried(s, colonist), colonistState(s, colonist))
	}
	if stored(s, storage) != before {
		t.Fatalf("the storage holds %d wood, want the colonist's %d", stored(s, storage), before)
	}
	if _, hasAxe := colonistAxe(s, colonist); !hasAxe {
		t.Fatal("the colonist put its axe away")
	}
	live(t, s, data.ColonistPlanEveryTicks+1, func() bool { return false })
	if state := colonistState(s, colonist); state != inputsources.Colonist_Idle {
		t.Fatalf("the colonist is %v with nothing left to put away", state)
	}
}

func TestStoringComesAfterFelling(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 4)
	harvest(s, tree)
	first := storageSpot(t, s, colonist, 10, 1, 1, tree)
	placeStorage(t, s, first, first)
	before := woodCarried(s, colonist)

	if !live(t, s, 20*data.TPS, func() bool { return !isTree(s, tree) }) {
		t.Fatalf("the colonist didn't fell the tree, it's %v", colonistState(s, colonist))
	}
	if woodCarried(s, colonist) != before {
		t.Fatal("the colonist put its wood away before felling")
	}
}

func TestStoringOnlyReachesSoFar(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := storageSpot(t, s, colonist, 6, data.StoreRangeTiles+2, 1)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(row))
	near := utils.Vec2i{X: row.X + data.StoreRangeTiles, Y: row.Y}
	far := utils.Vec2i{X: row.X + data.StoreRangeTiles + 1, Y: row.Y}
	farStorage := placeStorage(t, s, far, far)
	nearStorage := placeStorage(t, s, near, near)
	inv, _ := s.ECS.Inventories.Get(colonist)
	slot := -1
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok && countOf(s, item) > 1 {
			slot = i
		}
	}

	for _, storage := range []common.EntityId{farStorage, nearStorage} {
		inputs := map[common.EntityId]ecs.InputState{colonist: {Store: ecs.ItemStoring{Store: true, From: slot, Count: 1, Storage: storage}}}
		if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
			t.Fatal(err)
		}
	}
	if stored(s, farStorage) != 0 || stored(s, nearStorage) == 0 {
		t.Fatal("storing went into the storage out of reach, or not into the one in reach")
	}
}

func colonistAxe(s *testutil.Session, colonist common.EntityId) (common.EntityId, bool) {
	inv, _ := s.ECS.Inventories.Get(colonist)
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok {
			if eq, ok := s.ECS.Equipments.Get(item); ok && eq.HasAbility(abilities.Ability_Slash) {
				return item, true
			}
		}
	}
	return -1, false
}
