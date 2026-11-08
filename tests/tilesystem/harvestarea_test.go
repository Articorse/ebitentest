package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

// A square of harvest area on open ground away from the colonist, with a tree in its middle
func harvestSquare(t *testing.T, s *testutil.Session, from utils.Vec2i, side int) (tree utils.Vec2i, tiles []utils.Vec2i) {
	t.Helper()
	for dy := range side {
		for dx := range side {
			tiles = append(tiles, utils.Vec2i{X: from.X + dx, Y: from.Y + dy})
		}
	}
	harvest(s, tiles...)
	tree = utils.Vec2i{X: from.X + side/2, Y: from.Y + side/2}
	if !s.Chunks.PlaceFeature(tree, data.Feature_Tree) {
		t.Fatal("couldn't plant the tree")
	}
	return tree, tiles
}

func inHarvest(s *testutil.Session, tiles []utils.Vec2i) int {
	n := 0
	for _, tile := range tiles {
		if s.Chunks.InArea(data.Area_Harvest, tile) {
			n++
		}
	}
	return n
}

func fell(t *testing.T, s *testutil.Session, tree utils.Vec2i) {
	t.Helper()
	if destroyed, _ := s.Chunks.DamageFeature(tree, 1_000_000); !destroyed {
		t.Fatal("the tree wasn't felled")
	}
}

func TestTreesAreHarvestableAndDropWhatsGathered(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree, _ := harvestSquare(t, s, storageSpot(t, s, colonist, 6, 3, 3), 3)
	if !s.Chunks.IsHarvestable(tree) {
		t.Fatal("a tree isn't harvestable")
	}
	fell(t, s, tree)
	if s.Chunks.IsHarvestable(tree) {
		t.Fatal("a stump is harvestable")
	}
	if !s.Chunks.IsHarvestItem(data.Item_Wood) || s.Chunks.IsHarvestItem(data.Item_Torch) {
		t.Fatal("want wood to be what's harvested, and torches not")
	}
}

func TestAHarvestAreaGoesOnceNothingHarvestableOrHarvestedIsLeftInIt(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree, tiles := harvestSquare(t, s, storageSpot(t, s, colonist, 6, 3, 3), 3)
	corner := tiles[0]
	s.Chunks.SetGroundStack(corner, groundTorches(2))

	fell(t, s, tree)
	s.Chunks.SetGroundStack(tree, groundWood(2))
	s.ECS.ClearExhaustedHarvest()
	if inHarvest(s, tiles) != len(tiles) {
		t.Fatal("the harvest area went while the felled tree's wood was still lying in it")
	}

	s.Chunks.SetGroundStack(tree, groundWood(1))
	s.ECS.ClearExhaustedHarvest()
	if inHarvest(s, tiles) != len(tiles) {
		t.Fatal("the harvest area went with some of the wood still lying in it")
	}
	s.Chunks.SetGroundStack(tree, ecs.GroundStack{})
	s.ECS.ClearExhaustedHarvest()
	if n := inHarvest(s, tiles); n != 0 {
		t.Fatalf("%d tiles of the exhausted harvest area are left, with only torches lying in it", n)
	}
}

func TestOnlyTheExhaustedPartOfTheHarvestAreaGoes(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	corner := storageSpot(t, s, colonist, 6, 7, 4)
	exhausted, these := harvestSquare(t, s, corner, 3)
	_, apart := harvestSquare(t, s, utils.Vec2i{X: corner.X + 4, Y: corner.Y}, 3)
	// Touching the first only at a corner, and still with wood on it
	diagonal := utils.Vec2i{X: corner.X + 3, Y: corner.Y + 3}
	harvest(s, diagonal)
	looseWood(t, s, diagonal, 1)

	fell(t, s, exhausted)
	s.ECS.ClearExhaustedHarvest()
	if inHarvest(s, these) != 0 || inHarvest(s, apart) != len(apart) {
		t.Fatalf("%d of the exhausted area and %d of the one apart from it are left, want none and all %d", inHarvest(s, these), inHarvest(s, apart), len(apart))
	}
	if !s.Chunks.InArea(data.Area_Harvest, diagonal) {
		t.Fatal("the harvest area touching the exhausted one only at a corner went with it")
	}
}

func TestAHarvestAreaMarkedOnEmptyGroundStays(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	corner := storageSpot(t, s, colonist, 6, 3, 3)
	tiles := []utils.Vec2i{}
	for dy := range 3 {
		for dx := range 3 {
			tiles = append(tiles, utils.Vec2i{X: corner.X + dx, Y: corner.Y + dy})
		}
	}
	harvest(s, tiles...)
	s.Chunks.SetGroundStack(tiles[0], groundTorches(2))
	s.Chunks.SetGroundStack(tiles[0], ecs.GroundStack{})
	live(t, s, 2*data.TPS, func() bool { return false })
	if inHarvest(s, tiles) != len(tiles) {
		t.Fatal("a harvest area on empty ground went without anything harvested from it")
	}
}

func TestAColonistHarvestingAllOfAnAreaClearsIt(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	withoutWood(t, s, colonist)
	_, tiles := harvestSquare(t, s, storageSpot(t, s, colonist, 5, 3, 3), 3)
	if !live(t, s, 60*data.TPS, func() bool { return inHarvest(s, tiles) == 0 }) {
		t.Fatalf("%d tiles of the harvest area are left, the colonist is %v carrying %d wood", inHarvest(s, tiles), colonistState(s, colonist), woodCarried(s, colonist))
	}
}

func TestAHarvestAreaWithATreeLeftStays(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree, tiles := harvestSquare(t, s, storageSpot(t, s, colonist, 6, 3, 3), 3)
	if !s.Chunks.PlaceFeature(tiles[0], data.Feature_Tree) {
		t.Fatal("couldn't plant the other tree")
	}
	fell(t, s, tree)
	s.ECS.ClearExhaustedHarvest()
	if inHarvest(s, tiles) != len(tiles) {
		t.Fatal("the harvest area went with a tree still in it")
	}
}

func TestPickingUpNextToAHarvestAreaLeavesItAlone(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := storageSpot(t, s, colonist, 6, 4, 1)
	area := []utils.Vec2i{row, {X: row.X + 1, Y: row.Y}, {X: row.X + 2, Y: row.Y}}
	harvest(s, area...)
	next := utils.Vec2i{X: row.X + 3, Y: row.Y}
	looseWood(t, s, next, 2)
	s.Chunks.SetGroundStack(next, ecs.GroundStack{})
	s.ECS.ClearExhaustedHarvest()
	if inHarvest(s, area) != len(area) {
		t.Fatal("picking up next to an empty harvest area cleared it")
	}
}
