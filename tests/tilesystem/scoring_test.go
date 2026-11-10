package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

// Open ground other than the tile within the distance of it, closest first
func groundNear(t *testing.T, s *testutil.Session, near utils.Vec2i, tiles int) utils.Vec2i {
	t.Helper()
	best, found := utils.Vec2i{}, false
	for dy := -tiles; dy <= tiles; dy++ {
		for dx := -tiles; dx <= tiles; dx++ {
			tile := utils.Vec2i{X: near.X + dx, Y: near.Y + dy}
			if tile == near || utils.Dist2(tile, near) > tiles*tiles || !s.Chunks.CanPlaceSite(tile) {
				continue
			}
			if !found || utils.Dist2(tile, near) < utils.Dist2(best, near) {
				best, found = tile, true
			}
		}
	}
	if !found {
		t.Fatalf("there's no open ground within %d tiles of %v", tiles, near)
	}
	return best
}

func TestAfterFellingATreeAColonistGathersWhatLiesAroundItBeforeBuilding(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 5)
	harvest(s, tree)
	live(t, s, 1, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Chop {
		t.Fatalf("the colonist didn't go for the tree, it's %v", colonistState(s, colonist))
	}
	// Once it's there, so that it doesn't pick the wood up on its way
	stand := colonistJob(s, colonist).Stand
	if !live(t, s, 20*data.TPS, func() bool { return colonistTile(t, s, colonist) == stand }) {
		t.Fatal("the colonist didn't get to the tree")
	}

	lying := groundNear(t, s, tree, data.ColonistFollowUpRangeTiles)
	harvest(s, lying)
	looseWood(t, s, lying, 3*kg)
	site := groundAway(t, s, colonist, 3, tree, lying)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	s.Chunks.DeliverToSite(site, data.Item_Wood, wallCost(t))

	if !live(t, s, 20*data.TPS, func() bool { return !isTree(s, tree) }) {
		t.Fatal("the colonist didn't fell the tree")
	}
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Gather || job.Last != inputsources.Colonist_Chop {
		t.Fatalf("after felling the tree the colonist is %v, want gathering", job.State)
	}
}

func TestAColonistGathersTheStacksAroundTheFirstInOneGo(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	withoutWood(t, s, colonist)
	row := openRow(t, s, colonist, 4)
	stacks := []utils.Vec2i{row, {X: row.X + 1, Y: row.Y}, {X: row.X + 3, Y: row.Y}}
	harvest(s, stacks...)
	for _, tile := range stacks {
		looseWood(t, s, tile, 4*kg)
	}
	r := data.ColonistGatherRangeTiles
	if corner := (utils.Vec2i{X: row.X + r, Y: row.Y + r}); s.Chunks.CanPlaceSite(corner) {
		harvest(s, corner)
		looseWood(t, s, corner, 1*kg)
	} else if corner := (utils.Vec2i{X: row.X + r, Y: row.Y - r}); s.Chunks.CanPlaceSite(corner) {
		harvest(s, corner)
		looseWood(t, s, corner, 1*kg)
	} else {
		t.Fatal("there's no open ground diagonally out of gathering range")
	}

	live(t, s, 1, func() bool { return false })
	job := colonistJob(s, colonist)
	if job.State != inputsources.Colonist_Gather || len(job.Stops) != 3 {
		t.Fatalf("the colonist is on %v, want picking up all three stacks", job.Stops)
	}
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) == 12*kg }) {
		t.Fatalf("the colonist carries %d wood, want all 12", woodCarried(s, colonist))
	}
}

func TestAColonistOnlyGathersAsManyStacksAsItCanCarry(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	withoutWood(t, s, colonist)
	row := openRow(t, s, colonist, 3)
	fits := data.ColonistMaxCarryMg - s.ECS.CarriedMass(colonist)
	stacks := []utils.Vec2i{row, {X: row.X + 1, Y: row.Y}, {X: row.X + 2, Y: row.Y}}
	harvest(s, stacks...)
	for _, tile := range stacks {
		looseWood(t, s, tile, fits/2+1)
	}

	live(t, s, 1, func() bool { return false })
	job := colonistJob(s, colonist)
	total := 0
	for _, stop := range job.Stops {
		total += stop.Amount
	}
	if len(job.Stops) != 2 || total != fits {
		t.Fatalf("the colonist is on %v, want picking up the %d it can carry off two stacks", job.Stops, fits)
	}
}

func TestAfterGatheringAColonistPutsItAwayBeforeFelling(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	withoutWood(t, s, colonist)
	tile := storageSpot(t, s, colonist, 8, 1, 1)
	markStorage(t, s, false, tile)
	lying := groundAway(t, s, colonist, 4, tile)
	harvest(s, lying)
	looseWood(t, s, lying, 4*kg)
	live(t, s, 1, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Gather {
		t.Fatalf("the colonist didn't go to gather, it's %v", colonistState(s, colonist))
	}

	tree := treeAway(t, s, colonist, 4, tile, lying)
	harvest(s, tree)
	if !live(t, s, 20*data.TPS, func() bool { return colonistState(s, colonist) != inputsources.Colonist_Gather }) {
		t.Fatal("the colonist didn't finish gathering")
	}
	if state := colonistState(s, colonist); state != inputsources.Colonist_Store {
		t.Fatalf("after gathering the colonist is %v, want putting it away", state)
	}
}

func TestAColonistBuildsASiteItSuppliedBeforeOneCloserThatsReady(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	site := groundAway(t, s, colonist, 8)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	live(t, s, 1, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Deliver {
		t.Fatalf("the colonist didn't go to deliver, it's %v", colonistState(s, colonist))
	}

	ready := groundAway(t, s, colonist, 3, site)
	s.Chunks.PlaceSite(ready, data.Blueprint_WoodWall)
	s.Chunks.DeliverToSite(ready, data.Item_Wood, wallCost(t))
	if !live(t, s, 20*data.TPS, func() bool { return colonistState(s, colonist) != inputsources.Colonist_Deliver }) {
		t.Fatal("the colonist didn't finish delivering")
	}
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Construct || job.Stops[0].Tile != site {
		t.Fatalf("after delivering the colonist is on %v, want building the site it delivered to", job.Stops)
	}
}

func TestAColonistKeepsToTheSameWorkOverSomethingWeightierALittleFurther(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 2, 2, 1)
	markStorage(t, s, false, tile, utils.Vec2i{X: tile.X + 1, Y: tile.Y})
	torches, _ := s.ECS.SpawnItemStack(data.Item_Torch, 3, utils.Vec2f{})
	inv, _ := s.ECS.Inventories.Get(colonist)
	if err := s.ECS.PutInInventory(colonist, inv.FirstFreeSlot(), torches); err != nil {
		t.Fatal(err)
	}
	live(t, s, 1, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Store {
		t.Fatalf("the colonist didn't go to put things away, it's %v", colonistState(s, colonist))
	}

	// Close enough that felling it scores as well as putting away, but not as well as putting away again
	tree := treeAway(t, s, colonist, 4, tile)
	harvest(s, tree)
	if !live(t, s, 20*data.TPS, func() bool { return colonistJob(s, colonist).Last == inputsources.Colonist_Store }) {
		t.Fatal("the colonist didn't put the first thing away")
	}
	if state := colonistState(s, colonist); state != inputsources.Colonist_Store {
		t.Fatalf("after putting one thing away the colonist is %v, want putting the other away", state)
	}
}

// Only the site it was supplying, which has gone off home, is built as what follows, not another site ready to build
func TestOnlyTheSiteJustSuppliedIsBuiltAsWhatFollows(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	site := groundAway(t, s, colonist, 8)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	live(t, s, 1, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Deliver {
		t.Fatalf("the colonist didn't go to deliver, it's %v", colonistState(s, colonist))
	}

	lying := groundAway(t, s, colonist, 2, site)
	harvest(s, lying)
	looseWood(t, s, lying, 2*kg)
	ready := groundAway(t, s, colonist, 13, site, lying)
	s.Chunks.PlaceSite(ready, data.Blueprint_WoodWall)
	s.Chunks.DeliverToSite(ready, data.Item_Wood, wallCost(t))
	s.ECS.MarkArea(data.Area_Home, site, site, false)
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.Last != inputsources.Colonist_Deliver || job.State != inputsources.Colonist_Gather {
		t.Fatalf("the colonist is %v after giving up delivering, want gathering what's close rather than building far away", job.State)
	}
}

// The site it fetched for counts as what it was doing, not the storage it fetched from
func TestAColonistBuildsWhatItFetchedForBeforeHaulingOffAnotherSite(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 4, 2, 1)
	markStorage(t, s, false, tile, utils.Vec2i{X: tile.X + 1, Y: tile.Y})
	woodOntoStorage(t, s, colonist, tile)
	site := groundAway(t, s, colonist, 8, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	live(t, s, 1, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Fetch {
		t.Fatalf("the colonist didn't go to fetch, it's %v", colonistState(s, colonist))
	}

	cluttered := groundNear(t, s, site, 3)
	if !live(t, s, 20*data.TPS, func() bool { return colonistJob(s, colonist).Stops[0].Action == inputsources.Stop_Deliver }) {
		t.Fatal("the colonist didn't pick up what it fetched")
	}
	s.Chunks.PlaceSite(cluttered, data.Blueprint_WoodWall)
	s.Chunks.SetGroundStack(cluttered, groundTorches(2))
	if !live(t, s, 20*data.TPS, func() bool { return colonistState(s, colonist) != inputsources.Colonist_Fetch }) {
		t.Fatal("the colonist didn't deliver what it fetched")
	}
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Construct || job.Stops[0].Tile != site {
		t.Fatalf("after fetching the colonist is %v on %v, want building the site it fetched for", job.State, job.Stops)
	}
}

func TestAColonistRemembersWhatItLastDidWhileIdle(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 5)
	harvest(s, tree)
	if !live(t, s, 20*data.TPS, func() bool { return !isTree(s, tree) }) {
		t.Fatal("the colonist didn't fell the tree")
	}
	r := data.GroundSpillRangeTiles
	for _, tile := range s.Chunks.GroundStacksIn(utils.Vec2i{X: tree.X - r, Y: tree.Y - r}, utils.Vec2i{X: tree.X + r, Y: tree.Y + r}) {
		s.Chunks.SetGroundStack(tile, ecs.GroundStack{})
	}
	live(t, s, 3*data.TPS, func() bool { return false })
	if state := colonistState(s, colonist); state != inputsources.Colonist_Idle {
		t.Fatalf("the colonist is %v, want idle with nothing to do", state)
	}

	lying := groundNear(t, s, tree, data.ColonistFollowUpRangeTiles)
	harvest(s, lying)
	looseWood(t, s, lying, 2*kg)
	site := groundAway(t, s, colonist, 4, tree, lying)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	s.Chunks.DeliverToSite(site, data.Item_Wood, wallCost(t))
	live(t, s, data.ColonistPlanEveryTicks+1, func() bool { return colonistState(s, colonist) != inputsources.Colonist_Idle })
	if state := colonistState(s, colonist); state != inputsources.Colonist_Gather {
		t.Fatalf("the colonist is %v, want gathering what lies by the tree it felled", state)
	}
}

// The closest as the crow flies is cheaper to walk to than one a little further behind a wall, which is looked at too
// and has to lose out to it
func TestAColonistDoesntSwapTheBestJobItFoundForAWorseOne(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	self, ok := spotWithWallEast(s, colonistTile(t, s, colonist), 6, 7)
	if !ok {
		t.Fatal("there's no open ground for a wall near the colonist")
	}
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(self))
	for y := self.Y - 8; y <= self.Y+8; y++ {
		s.Chunks.PlaceTile(utils.Vec2i{X: self.X + 5, Y: y}, data.Tile_WoodWall)
	}
	open, behind := utils.Vec2i{X: self.X - 6, Y: self.Y}, utils.Vec2i{X: self.X + 7, Y: self.Y}
	s.Chunks.PlaceSite(open, data.Blueprint_WoodWall)
	s.Chunks.PlaceSite(behind, data.Blueprint_WoodWall)

	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Deliver || job.Stops[0].Tile != open {
		t.Fatalf("the colonist is on %v, want delivering to %v in the open", job.Stops, open)
	}
}

// A tile with open ground from west tiles west to east tiles east of it, apart from where a wall goes 5 east of it,
// which has room along it
func spotWithWallEast(s *testutil.Session, start utils.Vec2i, west, east int) (utils.Vec2i, bool) {
	fits := func(self utils.Vec2i) bool {
		for x := self.X - west; x <= self.X+east; x++ {
			if t := (utils.Vec2i{X: x, Y: self.Y}); x != self.X+5 && !s.Chunks.CanPlaceSite(t) && t != self {
				return false
			}
		}
		for y := self.Y - 8; y <= self.Y+8; y++ {
			if !s.Chunks.CanPlaceTile(utils.Vec2i{X: self.X + 5, Y: y}) {
				return false
			}
		}
		return s.Chunks.WalkSpeedPercent(self) > 0
	}
	for r := 0; r <= 40; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if self := (utils.Vec2i{X: start.X + dx, Y: start.Y + dy}); max(utils.AbsInt(dx), utils.AbsInt(dy)) == r && fits(self) {
					return self, true
				}
			}
		}
	}
	return utils.Vec2i{}, false
}
