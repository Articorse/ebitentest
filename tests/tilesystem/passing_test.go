package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

const passingRow = 14

// The colonist at the start of a row of open ground, on its way to fell a tree at the other end, past the middle
func onItsWayToATree(t *testing.T) (s *testutil.Session, colony, colonist common.EntityId, row []utils.Vec2i) {
	t.Helper()
	s, colony, colonist = colonistAtWork(t)
	row = rowOf(storageSpot(t, s, colonist, 4, passingRow, 1), passingRow)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(row[0]))
	tree := row[passingRow-1]
	if !s.Chunks.PlaceFeature(tree, data.Feature_Tree) {
		t.Fatal("couldn't plant the tree")
	}
	harvest(s, tree)
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Chop || job.Stops[0].Tile != tree {
		t.Fatalf("the colonist is on %v, want felling the tree", job.Stops)
	}
	return s, colony, colonist, row
}

// Until it's at the tree, and whether it ever stopped felling it
func walkToTheTree(t *testing.T, s *testutil.Session, colonist common.EntityId) bool {
	t.Helper()
	stand := colonistJob(s, colonist).Stand
	kept := true
	if !live(t, s, 20*data.TPS, func() bool {
		kept = kept && colonistState(s, colonist) == inputsources.Colonist_Chop
		return colonistTile(t, s, colonist) == stand
	}) {
		t.Fatal("the colonist didn't get to the tree")
	}
	return kept
}

func TestAColonistPicksUpWhatItPassesWithoutStopping(t *testing.T) {
	s, _, colonist, row := onItsWayToATree(t)
	middle := row[passingRow/2]
	harvest(s, middle)
	looseWood(t, s, middle, 3*kg)
	before := woodCarried(s, colonist)

	if !walkToTheTree(t, s, colonist) {
		t.Fatal("the colonist stopped felling the tree to pick something up")
	}
	if _, ok := s.Chunks.GroundStack(middle); ok || woodCarried(s, colonist) != before+3*kg {
		t.Fatalf("the colonist carries %d more wood, want the 3 it passed", woodCarried(s, colonist)-before)
	}
	if job := colonistJob(s, colonist); job.InPassing.Action != inputsources.Stop_PickUp || job.InPassing.Tile != middle {
		t.Fatalf("the colonist last did %v in passing, want picking up off %v", job.InPassing, middle)
	}
}

func TestAColonistClearsASiteOnHomeItPassesWhenThereIsRoomOnStorage(t *testing.T) {
	for _, c := range []struct {
		name                     string
		storage, offHome, hauled bool
		cleared                  bool
	}{
		{"with storage", true, false, false, true},
		{"without storage", false, false, false, false},
		{"off home", true, true, false, false},
		{"being hauled", true, false, true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, colony, colonist, row := onItsWayToATree(t)
			middle := row[passingRow/2]
			if c.storage {
				markStorage(t, s, false, groundAway(t, s, colonist, 6, row...))
			}
			s.Chunks.PlaceSite(middle, data.Blueprint_WoodWall)
			s.Chunks.SetGroundStack(middle, groundTorches(2))
			if c.offHome {
				s.ECS.MarkArea(data.Area_Home, middle, middle, false)
			}
			if c.hauled {
				other := anotherColonist(t, s, colony, colonist)
				testutil.SetPos(t, s.ECS, other, utils.TilePosToWorldPos(groundAway(t, s, colonist, 12, row...)))
				live(t, s, 1, func() bool { return false })
				if colonistState(s, other) != inputsources.Colonist_Haul {
					t.Fatalf("the other colonist is %v, want hauling", colonistState(s, other))
				}
			}

			walkToTheTree(t, s, colonist)
			if _, ok := s.Chunks.GroundStack(middle); ok == c.cleared {
				t.Fatalf("the torches on the site were left %v, want them cleared %v", ok, c.cleared)
			}
		})
	}
}

func TestAColonistPicksUpAnythingOnTheHarvestAreaButNotStorage(t *testing.T) {
	for _, onStorage := range []bool{false, true} {
		s, _, colonist, row := onItsWayToATree(t)
		middle := row[passingRow/2]
		harvest(s, middle)
		if onStorage {
			markStorage(t, s, false, middle)
		}
		s.Chunks.SetGroundStack(middle, groundTorches(2))

		walkToTheTree(t, s, colonist)
		if _, ok := s.Chunks.GroundStack(middle); ok != onStorage {
			t.Fatalf("on storage %v, the torches were left %v", onStorage, ok)
		}
	}
}

func TestAColonistLeavesWhatAnotherIsGoingToPickUp(t *testing.T) {
	s, colony, colonist, row := onItsWayToATree(t)
	middle := row[passingRow/2]
	harvest(s, middle)
	looseWood(t, s, middle, 3*kg)
	other := anotherColonist(t, s, colony, colonist)
	testutil.SetPos(t, s.ECS, other, utils.TilePosToWorldPos(groundAway(t, s, colonist, 12, row...)))
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, other); job.State != inputsources.Colonist_Gather || job.Stops[0].Tile != middle {
		t.Fatalf("the other colonist is on %v, want gathering the wood", job.Stops)
	}
	before := woodCarried(s, colonist)

	walkToTheTree(t, s, colonist)
	if woodCarried(s, colonist) != before {
		t.Fatal("the colonist picked up what the other was going to")
	}
}

func TestAColonistLeavesRoomForWhatItsJobIsToPickUp(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := rowOf(storageSpot(t, s, colonist, 4, passingRow, 1), passingRow)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(row[0]))
	tile := row[passingRow-1]
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	roomForWood(t, s, colonist, wallCost(t))
	site := groundAway(t, s, colonist, 6, row...)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	live(t, s, 1, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Fetch {
		t.Fatalf("the colonist didn't go to fetch, it's %v", colonistState(s, colonist))
	}

	middle := row[passingRow/2]
	harvest(s, middle)
	looseWood(t, s, middle, 1*kg)
	if !live(t, s, 20*data.TPS, func() bool { return colonistJob(s, colonist).Stops[0].Action == inputsources.Stop_Deliver }) {
		t.Fatal("the colonist didn't pick up what it was fetching")
	}
	if _, n := onGround(s, middle); n != 1*kg || woodCarried(s, colonist) != wallCost(t) {
		t.Fatalf("the colonist carries %d wood, want just the %d it fetched, leaving what it passed", woodCarried(s, colonist), wallCost(t))
	}
}

func TestAColonistDeliversWhatItDoesntNeedToASiteItPasses(t *testing.T) {
	s, _, colonist, row := onItsWayToATree(t)
	middle := row[passingRow/2]
	s.Chunks.PlaceSite(middle, data.Blueprint_WoodWall)
	before := woodCarried(s, colonist)

	if !walkToTheTree(t, s, colonist) {
		t.Fatal("the colonist stopped felling the tree to deliver")
	}
	if built, _ := s.Chunks.Site(middle); built.IsBlueprint() || woodCarried(s, colonist) != before-wallCost(t) {
		t.Fatal("the colonist didn't deliver to the site it passed")
	}
}

func TestAColonistDoesntDeliverInPassingToSitesOffHomeOrWhatItsJobNeeds(t *testing.T) {
	t.Run("off home", func(t *testing.T) {
		s, _, colonist, row := onItsWayToATree(t)
		middle := row[passingRow/2]
		s.Chunks.PlaceSite(middle, data.Blueprint_WoodWall)
		s.ECS.MarkArea(data.Area_Home, middle, middle, false)
		walkToTheTree(t, s, colonist)
		if built, _ := s.Chunks.Site(middle); !built.IsBlueprint() {
			t.Fatal("the colonist delivered to a site off home")
		}
	})
	t.Run("for its job", func(t *testing.T) {
		s, _, colonist := colonistAtWork(t)
		withoutWood(t, s, colonist)
		wood, _ := s.ECS.SpawnItemStack(data.Item_Wood, wallCost(t), utils.Vec2f{})
		if err := s.ECS.PutInInventory(colonist, 0, wood); err != nil {
			t.Fatal(err)
		}
		row := rowOf(storageSpot(t, s, colonist, 4, passingRow, 1), passingRow)
		testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(row[0]))
		far := row[passingRow-1]
		s.Chunks.PlaceSite(far, data.Blueprint_WoodWall)
		live(t, s, 1, func() bool { return false })
		if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Deliver || job.Stops[0].Tile != far {
			t.Fatalf("the colonist is on %v, want delivering to the far site", job.Stops)
		}

		middle := row[passingRow/2]
		s.Chunks.PlaceSite(middle, data.Blueprint_WoodWall)
		if !live(t, s, 20*data.TPS, func() bool { built, _ := s.Chunks.Site(far); return !built.IsBlueprint() }) {
			t.Fatal("the colonist didn't deliver to the far site")
		}
		if built, _ := s.Chunks.Site(middle); !built.IsBlueprint() {
			t.Fatal("the colonist delivered what its job needed to the site it passed")
		}
	})
}

func TestAColonistDoesNothingInPassingWhileWorking(t *testing.T) {
	s, _, colonist, row := onItsWayToATree(t)
	walkToTheTree(t, s, colonist)
	near := groundNear(t, s, colonistTile(t, s, colonist), 2)
	s.Chunks.PlaceSite(near, data.Blueprint_WoodWall)

	tree := row[passingRow-1]
	live(t, s, 20*data.TPS, func() bool {
		if built, _ := s.Chunks.Site(near); isTree(s, tree) && !built.IsBlueprint() {
			t.Fatal("the colonist delivered while felling the tree")
		}
		return !isTree(s, tree)
	})
}
