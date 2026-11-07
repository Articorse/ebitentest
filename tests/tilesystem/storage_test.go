package tilesystem_test

import (
	"ebittest/content/abilities"
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
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

func rect(first, last utils.Vec2i) []utils.Vec2i {
	tiles := []utils.Vec2i{}
	for y := first.Y; y <= last.Y; y++ {
		for x := first.X; x <= last.X; x++ {
			tiles = append(tiles, utils.Vec2i{X: x, Y: y})
		}
	}
	return tiles
}

func markStorage(t *testing.T, s *testutil.Session, erase bool, tiles ...utils.Vec2i) {
	t.Helper()
	handleInput(t, s, s.Player, ecs.InputState{Storage: ecs.StorageMarking{Mark: true, Erase: erase, Tiles: tiles}})
}

// How many of the kind lie on storage from first to last
func stored(s *testutil.Session, kind data.ItemEnum, first, last utils.Vec2i) int {
	n := 0
	for _, t := range s.Chunks.AreaTiles(data.Area_Storage, first, last) {
		if stack, ok := s.Chunks.GroundStack(t); ok && stack.Unique == nil && stack.Kind == kind {
			n += stack.Count
		}
	}
	return n
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

func TestStorageOnlyGoesOnOpenGroundInTheColonyOffSites(t *testing.T) {
	s, colony, colonist := colonistAtWork(t)
	open := storageSpot(t, s, colonist, 6, 1, 1)
	site := groundAway(t, s, colonist, 10, open)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)
	tree := treeAway(t, s, colonist, 12, open, site)
	wall := tileOf(t, s, data.Tile_Rock)
	cFirst, _, _ := s.ECS.ColonyBounds(colony)
	outside := utils.Vec2i{X: cFirst.X - 1, Y: cFirst.Y}

	markStorage(t, s, false, open, site, tree, wall, outside)
	if !s.Chunks.InArea(data.Area_Storage, open) {
		t.Fatal("storage wasn't marked out on open ground")
	}
	for _, c := range []struct {
		what string
		tile utils.Vec2i
	}{{"a site", site}, {"a tree", tree}, {"a wall", wall}, {"outside the colony", outside}} {
		if s.Chunks.InArea(data.Area_Storage, c.tile) {
			t.Fatalf("storage was marked out on %s", c.what)
		}
	}
	if s.ECS.CanPlaceStorage(site) || !s.ECS.CanPlaceStorage(utils.Vec2i{X: open.X, Y: open.Y}) {
		t.Fatal("CanPlaceStorage doesn't say what marking does")
	}
}

func TestErasingStorageTakesItOffJustThoseTilesAndLeavesWhatLiesThere(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 6, 3, 1)
	tiles := rect(first, utils.Vec2i{X: first.X + 2, Y: first.Y})
	markStorage(t, s, false, tiles...)
	s.ECS.LayDownKind(data.Item_Wood, 5, tiles[1])

	markStorage(t, s, true, tiles[1])
	if s.Chunks.InArea(data.Area_Storage, tiles[1]) || !s.Chunks.InArea(data.Area_Storage, tiles[0]) || !s.Chunks.InArea(data.Area_Storage, tiles[2]) {
		t.Fatal("erasing the middle of storage didn't take just it off")
	}
	if kind, n := onGround(s, tiles[1]); kind != data.Item_Wood || n != 5 {
		t.Fatal("erasing storage took away what lay on it")
	}
}

func TestNothingIsBuiltOrPlacedOnStorage(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)

	if s.Chunks.CanPlaceSite(tile) || s.Chunks.CanPlaceFeature(tile) || s.Chunks.PlaceFeature(tile, data.Feature_Torch) {
		t.Fatal("a blueprint or a feature can go on storage")
	}
	handleInput(t, s, s.Player, ecs.InputState{PlaceBlueprint: ecs.BlueprintPlacement{Kind: data.Blueprint_Planks, Tiles: []utils.Vec2i{tile}}})
	if _, ok := s.Chunks.Site(tile); ok {
		t.Fatal("a blueprint went on storage")
	}
}

func TestStorageIsSavedWithItsChunk(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)
	s.Commit()
	if r := testutil.StartSession(t); !r.Chunks.InArea(data.Area_Storage, tile) {
		t.Fatal("storage wasn't saved")
	}
}

func TestAColonistPutsAwayWhatItCarriesButItsAxeOnOneTile(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 10, 3, 1)
	last := utils.Vec2i{X: first.X + 2, Y: first.Y}
	markStorage(t, s, false, rect(first, last)...)
	before := woodCarried(s, colonist)

	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) == 0 }) {
		t.Fatalf("the colonist still carries %d wood, it's %v", woodCarried(s, colonist), colonistState(s, colonist))
	}
	if stored(s, data.Item_Wood, first, last) != before || len(s.Chunks.GroundStacksIn(first, last)) != 1 {
		t.Fatalf("storage holds %d wood on %d tiles, want the colonist's %d on one", stored(s, data.Item_Wood, first, last), len(s.Chunks.GroundStacksIn(first, last)), before)
	}
	if _, hasAxe := colonistAxe(s, colonist); !hasAxe {
		t.Fatal("the colonist put its axe away")
	}
	live(t, s, data.ColonistPlanEveryTicks+1, func() bool { return false })
	if state := colonistState(s, colonist); state != inputsources.Colonist_Idle {
		t.Fatalf("the colonist is %v with nothing left to put away", state)
	}
}

func TestAColonistPutsAwayWithWhatsAlreadyThereBeforeUsingEmptyStorage(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 10, 3, 1)
	last := utils.Vec2i{X: first.X + 2, Y: first.Y}
	markStorage(t, s, false, rect(first, last)...)
	s.ECS.LayDownKind(data.Item_Wood, 50, last)
	before := woodCarried(s, colonist)

	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) == 0 }) {
		t.Fatalf("the colonist didn't put its wood away, it's %v", colonistState(s, colonist))
	}
	if _, n := onGround(s, last); n != 50+before || len(s.Chunks.GroundStacksIn(first, last)) != 1 {
		t.Fatalf("the stack already on storage holds %d, want the colonist's %d added to its 50", n, before)
	}
}

func TestStoringComesAfterFelling(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 4)
	harvest(s, tree)
	tile := storageSpot(t, s, colonist, 10, 1, 1, tree)
	markStorage(t, s, false, tile)
	before := woodCarried(s, colonist)

	if !live(t, s, 20*data.TPS, func() bool { return !isTree(s, tree) }) {
		t.Fatalf("the colonist didn't fell the tree, it's %v", colonistState(s, colonist))
	}
	if woodCarried(s, colonist) != before {
		t.Fatal("the colonist put its wood away before felling")
	}
}

func TestGatheringLeavesWhatsOnStorage(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 5, 1, 1)
	harvest(s, tile)
	markStorage(t, s, false, tile)
	s.ECS.LayDownKind(data.Item_Wood, 7, tile)
	inv, _ := s.ECS.Inventories.Get(colonist)
	wood, _ := inv.Slot(0)
	s.ECS.ScheduleRemoveEntity(wood)
	testutil.RemoveScheduled(t, s.ECS)

	live(t, s, 5*data.TPS, func() bool { return false })
	if _, n := onGround(s, tile); n != 7 {
		t.Fatal("the colonist gathered what was put away on storage")
	}
}

func TestPuttingDownOnlyGoesOnTheTileAskedForWithinReach(t *testing.T) {
	s, colonist, row := colonistHoldingWood(t)
	near := utils.Vec2i{X: row.X + data.ItemReachTiles, Y: row.Y}
	far := utils.Vec2i{X: row.X + data.ItemReachTiles + 1, Y: row.Y}
	other := utils.Vec2i{X: row.X + 1, Y: row.Y}
	s.ECS.LayDownKind(data.Item_Torch, 1, other)
	before := woodCarried(s, colonist)

	for _, tile := range []utils.Vec2i{far, other, near} {
		handleInput(t, s, colonist, ecs.InputState{PutDown: ecs.ItemPutDown{PutDown: true, From: 0, Count: 3, Tile: tile}})
	}
	if _, ok := s.Chunks.GroundStack(far); ok {
		t.Fatal("wood was put down out of reach")
	}
	if kind, _ := onGround(s, other); kind != data.Item_Torch {
		t.Fatal("wood was put down on a torch")
	}
	if kind, n := onGround(s, near); kind != data.Item_Wood || n != 3 || woodCarried(s, colonist) != before-3 {
		t.Fatalf("putting down 3 wood in reach left %d of %v there and %d carried", n, kind, woodCarried(s, colonist))
	}
	if len(s.Chunks.GroundStacksIn(utils.Vec2i{X: row.X - 1, Y: row.Y - 1}, utils.Vec2i{X: far.X + 1, Y: row.Y + 1})) != 2 {
		t.Fatal("putting down spilled onto other tiles")
	}
}
