package tilesystem_test

import (
	"ebittest/content/abilities"
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

// A tree on ground the given distance from the colonist
func treeAway(t *testing.T, s *testutil.Session, colonist common.EntityId, tiles int, not ...utils.Vec2i) utils.Vec2i {
	t.Helper()
	tile := groundAway(t, s, colonist, tiles, not...)
	if !s.Chunks.PlaceFeature(tile, data.Feature_Tree) {
		t.Fatalf("couldn't plant a tree on %v", tile)
	}
	return tile
}

func isTree(s *testutil.Session, tile utils.Vec2i) bool {
	f, ok := s.Chunks.FeatureAt(tile)
	return ok && f == data.Feature_Tree
}

func harvest(s *testutil.Session, tiles ...utils.Vec2i) {
	for _, tile := range tiles {
		s.ECS.MarkArea(data.Area_Harvest, tile, tile, true)
	}
}

func TestAColonistFellsATreeOnTheHarvestArea(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 6)
	harvest(s, tree)

	if !live(t, s, 20*data.TPS, func() bool { return !isTree(s, tree) }) {
		t.Fatalf("the colonist didn't fell the tree, it's %v", colonistState(s, colonist))
	}
	if f, _ := s.Chunks.FeatureAt(tree); f != data.Feature_Stump {
		t.Fatalf("felling the tree left %v, want a stump", f)
	}
	live(t, s, data.ColonistPlanEveryTicks+1, func() bool { return false })
	if state := colonistState(s, colonist); state != inputsources.Colonist_Idle {
		t.Fatalf("the colonist is %v with nothing left to fell", state)
	}
}

func TestAColonistLeavesTreesOffTheHarvestAreaAlone(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 6)

	live(t, s, 5*data.TPS, func() bool { return false })
	if !isTree(s, tree) || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatalf("the colonist went for a tree on home that isn't to be harvested, it's %v", colonistState(s, colonist))
	}
}

func TestBuildingComesBeforeFelling(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 4)
	harvest(s, tree)
	site := groundAway(t, s, colonist, 10, tree)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)

	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the colonist didn't build the blueprint, it's %v", colonistState(s, colonist))
	}
	if !isTree(s, tree) {
		t.Fatal("the colonist felled the closer tree before building")
	}
	if !live(t, s, 20*data.TPS, func() bool { return !isTree(s, tree) }) {
		t.Fatal("the colonist didn't fell the tree once it had built")
	}
}

func TestAColonistStopsFellingATreeTakenOffTheHarvestArea(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 6)
	harvest(s, tree)
	if !live(t, s, 2*data.TPS, func() bool { return colonistState(s, colonist) == inputsources.Colonist_Chop }) {
		t.Fatalf("the colonist didn't go for the tree, it's %v", colonistState(s, colonist))
	}

	s.ECS.MarkArea(data.Area_Harvest, tree, tree, false)
	live(t, s, 1, func() bool { return false })
	if state := colonistState(s, colonist); state != inputsources.Colonist_Idle {
		t.Fatalf("the colonist is still %v after the tree was taken off the harvest area", state)
	}
	live(t, s, 10*data.TPS, func() bool { return false })
	if !isTree(s, tree) {
		t.Fatal("the colonist felled a tree taken off the harvest area")
	}
}

func TestAColonistWithoutAnAxeLeavesTreesAlone(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	inv, _ := s.ECS.Inventories.Get(colonist)
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok {
			if eq, ok := s.ECS.Equipments.Get(item); ok && eq.HasAbility(abilities.Ability_Slash) {
				s.ECS.ScheduleRemoveEntity(item)
			}
		}
	}
	if err := s.ECS.RemoveScheduledEntities(); err != nil {
		t.Fatal(err)
	}
	tree := treeAway(t, s, colonist, 6)
	harvest(s, tree)

	live(t, s, 5*data.TPS, func() bool { return false })
	if !isTree(s, tree) || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatalf("a colonist without an axe went for a tree, it's %v", colonistState(s, colonist))
	}
}

func TestTreesAreFoundInTileOrderAcrossChunks(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	first := utils.Vec2i{X: data.ChunkSize - 3, Y: -2}
	last := utils.Vec2i{X: data.ChunkSize + 2, Y: 2}
	planted := 0
	for y := first.Y - 1; y <= last.Y+1; y++ {
		for x := first.X - 1; x <= last.X+1; x++ {
			if (x+y)%3 == 0 && s.Chunks.PlaceFeature(utils.Vec2i{X: x, Y: y}, data.Feature_Tree) {
				planted++
			}
		}
	}
	if planted < 10 {
		t.Fatalf("only %d trees could be planted", planted)
	}
	torch := utils.Vec2i{X: data.ChunkSize, Y: 1}
	s.Chunks.PlaceFeature(torch, data.Feature_Torch)

	want := []utils.Vec2i{}
	for y := first.Y; y <= last.Y; y++ {
		for x := first.X; x <= last.X; x++ {
			if isTree(s, utils.Vec2i{X: x, Y: y}) {
				want = append(want, utils.Vec2i{X: x, Y: y})
			}
		}
	}
	if got := s.Chunks.FeaturesIn(first, last, data.Feature_Tree); !slices.Equal(got, want) {
		t.Fatalf("the trees found are %v, want %v", got, want)
	}
}
