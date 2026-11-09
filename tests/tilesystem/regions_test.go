package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

// The tiles r away from the middle, as a square, other than the gaps
func ring(middle utils.Vec2i, r int, gaps ...utils.Vec2i) []utils.Vec2i {
	tiles := []utils.Vec2i{}
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			t := at(middle, dx, dy)
			if max(abs(dx), abs(dy)) != r {
				continue
			}
			gap := false
			for _, g := range gaps {
				gap = gap || g == t
			}
			if !gap {
				tiles = append(tiles, t)
			}
		}
	}
	return tiles
}

func TestTilesWalledOffFromEachOtherArentReachable(t *testing.T) {
	for _, c := range []struct {
		name             string
		door             bool
		faction          data.FactionEnum
		colony, monsters bool
	}{
		{"without a door", false, data.Faction_None, false, false},
		{"with the colony's door", true, data.Faction_Colony, true, false},
		{"with no one's door", true, data.Faction_None, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, _, colonist := colonistAtWork(t)
			middle := doorYard(t, s, colonist)
			outside := colonistTile(t, s, colonist)
			gap := at(middle, 0, 3)
			if !s.Chunks.Reachable(middle, outside, data.Faction_Monsters, utils.Vec2i{}, utils.Vec2i{}) {
				t.Fatal("the middle of open ground can't be reached")
			}
			wall(t, s, ring(middle, 3, gap)...)
			if !s.Chunks.Reachable(middle, outside, data.Faction_Monsters, utils.Vec2i{}, utils.Vec2i{}) {
				t.Fatal("the middle can't be reached through the gap")
			}
			if c.door {
				buildDoor(t, s, gap, c.faction)
			} else {
				wall(t, s, gap)
			}

			if got := s.Chunks.Reachable(middle, outside, data.Faction_Colony, utils.Vec2i{}, utils.Vec2i{}); got != c.colony {
				t.Errorf("the colony reaches inside %v, want %v", got, c.colony)
			}
			if got := s.Chunks.Reachable(outside, middle, data.Faction_Monsters, utils.Vec2i{}, utils.Vec2i{}); got != c.monsters {
				t.Errorf("monsters reach inside %v, want %v", got, c.monsters)
			}
			if s.Chunks.Reachable(outside, middle, data.Faction_None, utils.Vec2i{}, utils.Vec2i{}) {
				t.Error("something of no faction reaches inside")
			}
			if c.door && (!s.Chunks.Reachable(gap, middle, data.Faction_Monsters, utils.Vec2i{}, utils.Vec2i{}) || !s.Chunks.Reachable(gap, outside, data.Faction_Monsters, utils.Vec2i{}, utils.Vec2i{})) {
				t.Error("a monster on the door can't get off it to either side")
			}

			if destroyed, _ := s.Chunks.DamageTile(at(middle, 3, 0), 1000); !destroyed {
				t.Fatal("the wall couldn't be knocked down")
			}
			if !s.Chunks.Reachable(outside, middle, data.Faction_Monsters, utils.Vec2i{}, utils.Vec2i{}) {
				t.Error("monsters can't reach inside through the knocked-down wall")
			}
		})
	}
}

func TestRegionsJoinAcrossChunkEdges(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	start := colonistTile(t, s, colonist)
	edge := (start.X/data.ChunkSize + 1) * data.ChunkSize
	for y := start.Y - 40; y <= start.Y+40; y++ {
		a, b := utils.Vec2i{X: edge - 1, Y: y}, utils.Vec2i{X: edge, Y: y}
		if s.Chunks.WalkSpeedPercent(a) == 0 || s.Chunks.WalkSpeedPercent(b) == 0 {
			continue
		}
		if !s.Chunks.Reachable(a, b, data.Faction_Monsters, utils.Vec2i{}, utils.Vec2i{}) {
			t.Fatalf("%v can't be reached from %v next to it across the chunk edge", b, a)
		}
		return
	}
	t.Fatal("no open ground on both sides of the chunk edge")
}

func TestAnEnclosureGoesUpToItsLimitAndOnlyThroughTheFactionsDoors(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	gap := at(middle, 0, 3)
	wall(t, s, ring(middle, 3, gap)...)
	far := colonistTile(t, s, colonist)

	for _, c := range []struct {
		name     string
		closed   utils.Vec2i
		limit    int
		tiles    int
		enclosed bool
	}{
		{"with the gap closed", gap, 1000, 25, true},
		{"with the gap open", far, 1000, 1000, false},
		{"up to a limit", gap, 10, 10, false},
	} {
		tiles, enclosed := s.Chunks.Enclosure(middle, c.closed, data.Faction_Colony, c.limit)
		if len(tiles) != c.tiles || enclosed != c.enclosed {
			t.Errorf("%s, the enclosure is %d tiles, enclosed %v; want %d, %v", c.name, len(tiles), enclosed, c.tiles, c.enclosed)
		}
	}

	buildDoor(t, s, gap, data.Faction_Colony)
	if _, enclosed := s.Chunks.Enclosure(middle, far, data.Faction_Colony, 1000); enclosed {
		t.Error("the colony is shut in by its own door")
	}
	if tiles, enclosed := s.Chunks.Enclosure(middle, far, data.Faction_Monsters, 1000); !enclosed || len(tiles) != 25 {
		t.Errorf("monsters are shut in %v, in %d tiles, by the colony's door; want shut in 25", enclosed, len(tiles))
	}
}

// The yard walled round with the gap left for a supplied wall site, and the 5×5 inside
func roomWithAWallToFinish(t *testing.T, s *testutil.Session, middle, gap utils.Vec2i) func(utils.Vec2i) bool {
	t.Helper()
	wall(t, s, ring(middle, 3, gap)...)
	if !s.Chunks.PlaceSite(gap, data.Blueprint_WoodWall) {
		t.Fatal("couldn't place the last wall")
	}
	s.Chunks.DeliverToSite(gap, data.Item_Wood, wallCost(t))
	return func(tile utils.Vec2i) bool { return abs(tile.X-middle.X) < 3 && abs(tile.Y-middle.Y) < 3 }
}

func TestABuilderFinishesAWallFromOutsideWhatItWouldShutIn(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	gap := at(middle, 0, 3)
	inside := roomWithAWallToFinish(t, s, middle, gap)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(middle))
	live(t, s, 1, func() bool { return false })
	if stand := colonistJob(s, colonist).Stand; inside(stand) {
		t.Fatalf("the colonist chose to build from %v, inside", stand)
	}

	if !live(t, s, 30*data.TPS, isBuilt(s, gap)) {
		t.Fatal("the colonist didn't finish the wall")
	}
	if inside(colonistTile(t, s, colonist)) {
		t.Fatalf("the colonist shut itself in, on %v", colonistTile(t, s, colonist))
	}
}

func TestABuilderShutInEitherWayFinishesWhereItIs(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	wall(t, s, ring(middle, 3)...)
	wall(t, s, at(middle, -2, 0), at(middle, -1, 0), at(middle, 1, 0), at(middle, 2, 0))
	if !s.Chunks.PlaceSite(middle, data.Blueprint_WoodWall) {
		t.Fatal("couldn't place the wall between the halves")
	}
	s.Chunks.DeliverToSite(middle, data.Item_Wood, wallCost(t))
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(middle, 0, -2)))

	if !live(t, s, 30*data.TPS, isBuilt(s, middle)) {
		t.Fatalf("the colonist didn't finish the wall, it's %v on %v", colonistState(s, colonist), colonistTile(t, s, colonist))
	}
}

func TestABuilderWaitsForAnotherColonistToGetOutOfWhatItWouldShutIn(t *testing.T) {
	s, colony, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	gap := at(middle, 0, 3)
	inside := roomWithAWallToFinish(t, s, middle, gap)
	live(t, s, 1, func() bool { return false })
	other := anotherColonist(t, s, colony, colonist)

	told := func() bool {
		em := colonistJob(s, other).Emergency
		return em.Kind == inputsources.Emergency_GetOut && em.Tile == gap
	}
	if !live(t, s, 10*data.TPS, func() bool {
		testutil.SetPos(t, s.ECS, other, utils.TilePosToWorldPos(middle))
		if isBuilt(s, gap)() {
			t.Fatal("the colonist shut the other one in")
		}
		return told()
	}) {
		t.Fatal("the other colonist wasn't told to get out of the way of the wall")
	}
	until := colonistJob(s, other).Emergency.Until
	live(t, s, data.TPS, func() bool {
		testutil.SetPos(t, s.ECS, other, utils.TilePosToWorldPos(middle))
		if isBuilt(s, gap)() {
			t.Fatal("the colonist shut the other one in")
		}
		return false
	})
	if em := colonistJob(s, other).Emergency; em.Until != until {
		t.Fatalf("being told again put off the end of the emergency from tick %d to %d", until, em.Until)
	}
	if !live(t, s, 30*data.TPS, isBuilt(s, gap)) {
		t.Fatal("the colonist didn't finish the wall once the other one could get out")
	}
	if inside(colonistTile(t, s, other)) {
		t.Fatal("the other colonist was shut in")
	}
	live(t, s, 1, func() bool { return false })
	if em := colonistJob(s, other).Emergency; em.Kind != inputsources.Emergency_None {
		t.Fatalf("the other colonist is still in an emergency, %+v, once the wall's built", em)
	}
}

func TestAColonistLooksPastSitesItCantGetTo(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	wall(t, s, ring(middle, 3)...)
	for dy := -1; dy <= 1; dy++ {
		for dx := -2; dx <= 2; dx += 2 {
			site := at(middle, dx, dy)
			if !s.Chunks.PlaceSite(site, data.Blueprint_WoodWall) {
				t.Fatalf("couldn't place a site on %v", site)
			}
			s.Chunks.DeliverToSite(site, data.Item_Wood, wallCost(t))
		}
	}
	reachable := groundAwayWhere(t, s, colonist, 25, func(tile utils.Vec2i) bool { return utils.Dist2(tile, middle) > 8*8 })
	if !s.Chunks.PlaceSite(reachable, data.Blueprint_WoodWall) {
		t.Fatal("couldn't place the site that can be got to")
	}
	s.Chunks.DeliverToSite(reachable, data.Item_Wood, wallCost(t))

	if !live(t, s, 40*data.TPS, isBuilt(s, reachable)) {
		t.Fatalf("the colonist didn't build the site it can get to past the %d it can't, it's %v", data.ColonistPathSearches+1, colonistState(s, colonist))
	}
}
