package tilesystem_test

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

// A giant colonist of the colony in the middle of 17 by 17 tiles of open ground, with the colonist at work disabled
func giantAtWork(t *testing.T) (s *testutil.Session, colony, giant common.EntityId, middle utils.Vec2i) {
	t.Helper()
	s, colony, colonist := colonistAtWork(t)
	middle = at(storageSpot(t, s, colonist, 20, 17, 17), 8, 8)
	s.ECS.SetDisabled(colonist, true)
	giant, err := templates.SpawnGiantColonist(s.ECS, colony, utils.TilePosToWorldPos(middle))
	if err != nil {
		t.Fatal(err)
	}
	return s, colony, giant, middle
}

func TestAGiantColonistBuildsFromWhereItsBodyIsOffTheSite(t *testing.T) {
	s, _, giant, middle := giantAtWork(t)
	site := at(middle, 4, 0)
	if !s.Chunks.PlaceSite(site, data.Blueprint_WoodWall) {
		t.Fatal("couldn't place the site")
	}
	s.Chunks.DeliverToSite(site, data.Item_Wood, wallCost(t))
	live(t, s, 1, func() bool { return false })
	if stand := colonistJob(s, giant).Stand; slices.Contains(s.ECS.BodyTiles(giant, stand), site) {
		t.Fatalf("the giant colonist chose to stand on %v, with its body over the site", stand)
	}
	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the giant colonist didn't build the wall, it's %v on %v", colonistState(s, giant), colonistTile(t, s, giant))
	}
}

func TestAGiantColonistFellsATreeFromTheEdgeOfItsBody(t *testing.T) {
	for _, c := range []struct {
		name   string
		dx, dy int
	}{{"to the right", 4, 0}, {"above and to the left", -4, -3}} {
		t.Run(c.name, func(t *testing.T) {
			s, _, giant, middle := giantAtWork(t)
			tree := at(middle, c.dx, c.dy)
			if !s.Chunks.PlaceFeature(tree, data.Feature_Tree) {
				t.Fatal("couldn't plant the tree")
			}
			harvest(s, tree)
			if !live(t, s, 60*data.TPS, func() bool { return !isTree(s, tree) }) {
				t.Fatalf("the giant colonist didn't fell the tree, it's %v on %v", colonistState(s, giant), colonistTile(t, s, giant))
			}
		})
	}
}

func TestAGiantColonistOpensADoorItsBodyIsNextTo(t *testing.T) {
	s, _, giant, middle := giantAtWork(t)
	doorTile := at(middle, 0, -3)
	wall(t, s, at(doorTile, -1, 0), at(doorTile, 1, 0))
	buildDoor(t, s, doorTile, data.Faction_Colony)
	s.ECS.SetDisabled(giant, true)
	s.Chunks.TickDoors(s.ECS)
	if door(t, s, doorTile).Open {
		t.Fatal("the door opened with no one by it")
	}
	s.ECS.SetDisabled(giant, false)
	testutil.SetPos(t, s.ECS, giant, utils.TilePosToWorldPos(at(doorTile, 0, 2)))
	s.Chunks.TickDoors(s.ECS)
	if !door(t, s, doorTile).Open {
		t.Fatal("the door didn't open for the giant colonist whose body reaches next to it")
	}
}

func TestTwoGiantColonistsDontChooseStandsTheirBodiesWouldOverlapOn(t *testing.T) {
	s, colony, giant, middle := giantAtWork(t)
	testutil.SetPos(t, s.ECS, giant, utils.TilePosToWorldPos(at(middle, -6, -2)))
	other, err := templates.SpawnGiantColonist(s.ECS, colony, utils.TilePosToWorldPos(at(middle, -1, 3)))
	if err != nil {
		t.Fatal(err)
	}
	for _, site := range []utils.Vec2i{at(middle, 6, -2), at(middle, 6, 2)} {
		if !s.Chunks.PlaceSite(site, data.Blueprint_WoodWall) {
			t.Fatal("couldn't place a site")
		}
		s.Chunks.DeliverToSite(site, data.Item_Wood, wallCost(t))
	}
	live(t, s, 1, func() bool { return false })
	a, b := colonistJob(s, giant), colonistJob(s, other)
	if a.State != b.State || a.State == 0 {
		t.Fatalf("the giant colonists are %v and %v, want both building", a.State, b.State)
	}
	if body := s.ECS.BodyTiles(giant, a.Stand); slices.ContainsFunc(s.ECS.BodyTiles(other, b.Stand), func(t utils.Vec2i) bool { return slices.Contains(body, t) }) {
		t.Fatalf("the giant colonists chose stands %v and %v, where their bodies overlap", a.Stand, b.Stand)
	}
}

func TestAGiantColonistIsShutInWhereOnlyAColonistOfATileCouldGetOut(t *testing.T) {
	s, colony, giant, middle := giantAtWork(t)
	gap := at(middle, 0, 3)
	wall(t, s, ring(middle, 3, gap)...)
	far := at(middle, 8, 8)
	tiles, enclosed := s.ECS.Enclosure(giant, middle, far, data.ColonistEnclosedTiles)
	if !enclosed || len(tiles) != 2 {
		t.Fatalf("the giant colonist is shut in %v, in %d tiles, want shut in the 2 in the middle it fits on", enclosed, len(tiles))
	}
	small, err := templates.SpawnColonist(s.ECS, colony)
	if err != nil {
		t.Fatal(err)
	}
	if _, enclosed := s.ECS.Enclosure(small, middle, far, data.ColonistEnclosedTiles); enclosed {
		t.Fatal("a colonist of a tile is shut in, with the gap to get out by")
	}
}

func TestAnIdleGiantColonistStepsItsBodyOffASite(t *testing.T) {
	s, _, giant, middle := giantAtWork(t)
	site := at(middle, 1, 0)
	if !s.Chunks.PlaceSite(site, data.Blueprint_WoodWall) {
		t.Fatal("couldn't place the site")
	}
	s.ECS.MarkArea(data.Area_Home, site, site, false)
	if !live(t, s, 10*data.TPS, func() bool {
		return !slices.Contains(s.ECS.Footprint(giant), site) && !slices.Contains(s.ECS.BodyTiles(giant, colonistTile(t, s, giant)), site)
	}) {
		t.Fatalf("the giant colonist stayed with its body over the site, on %v", colonistTile(t, s, giant))
	}
}
