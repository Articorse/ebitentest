package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"fmt"
	"slices"
	"testing"
)

// Another colonist of the colony, standing where the first one is
func anotherColonist(t *testing.T, s *testutil.Session, colony, colonist common.EntityId) common.EntityId {
	t.Helper()
	other, err := templates.SpawnColonist(s.ECS, colony)
	if err != nil {
		t.Fatal(err)
	}
	testutil.SetPos(t, s.ECS, other, testutil.WorldPos(t, s.ECS, colonist))
	return other
}

func colonistJob(s *testutil.Session, colonist common.EntityId) *inputsources.InputColonistParams {
	in, _ := s.ECS.Inputs.Get(colonist)
	return in.Params().(*inputsources.InputColonistParams)
}

func TestTwoColonistsFellDifferentTrees(t *testing.T) {
	s, colony, a := colonistAtWork(t)
	b := anotherColonist(t, s, colony, a)
	near := treeAway(t, s, a, 4)
	far := treeAway(t, s, a, 9, near)
	harvest(s, near, far)

	live(t, s, 2, func() bool { return false })
	ja, jb := colonistJob(s, a), colonistJob(s, b)
	if ja.State != inputsources.Colonist_Chop || jb.State != inputsources.Colonist_Chop || ja.Stops[0].Tile == jb.Stops[0].Tile {
		t.Fatalf("the colonists are %v and %v, want each felling a tree of its own", ja.Stops, jb.Stops)
	}
	if !live(t, s, 30*data.TPS, func() bool { return !isTree(s, near) && !isTree(s, far) }) {
		t.Fatal("the colonists didn't fell both trees")
	}
}

func TestTwoColonistsGatherAStackBetweenThemByWhatTheyCanCarry(t *testing.T) {
	s, colony, a := colonistAtWork(t)
	b := anotherColonist(t, s, colony, a)
	withoutWood(t, s, a)
	withoutWood(t, s, b)
	lying := groundAway(t, s, a, 5)
	harvest(s, lying)
	fits := (data.ColonistMaxCarryWeight - s.ECS.CarriedWeight(a)) / 2
	looseWood(t, s, lying, fits+5)

	live(t, s, 2, func() bool { return false })
	ja, jb := colonistJob(s, a), colonistJob(s, b)
	if len(ja.Stops) != 1 || len(jb.Stops) != 1 || ja.Stops[0].Count+jb.Stops[0].Count != fits+5 {
		t.Fatalf("the colonists set out to pick up %v and %v, want all %d between them", ja.Stops, jb.Stops, fits+5)
	}
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, a)+woodCarried(s, b) == fits+5 }) {
		t.Fatalf("the colonists carry %d and %d wood, want the %d between them", woodCarried(s, a), woodCarried(s, b), fits+5)
	}
}

// The user's case: a colonist carrying wood leaves a site alone that another is fetching for, even though it got there
// first
func TestASiteBeingFetchedForIsLeftToTheColonistFetching(t *testing.T) {
	s, colony, fetcher := colonistAtWork(t)
	tile := storageSpot(t, s, fetcher, 12, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, fetcher, tile)
	site := groundAway(t, s, fetcher, 5, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	live(t, s, 2, func() bool { return false })
	if colonistState(s, fetcher) != inputsources.Colonist_Fetch {
		t.Fatalf("the colonist didn't go to fetch, it's %v", colonistState(s, fetcher))
	}

	carrier := anotherColonist(t, s, colony, fetcher)
	delivered := false
	if !live(t, s, 30*data.TPS, func() bool {
		delivered = delivered || colonistState(s, carrier) == inputsources.Colonist_Deliver
		return isBuilt(s, site)()
	}) {
		t.Fatalf("the site wasn't built, the colonists are %v and %v", colonistState(s, fetcher), colonistState(s, carrier))
	}
	if delivered {
		t.Fatal("the colonist carrying wood went to deliver to the site the other was fetching for")
	}
}

func TestAColonistFinishesFellingBeforeTurningToABlueprint(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 6)
	harvest(s, tree)
	live(t, s, 2, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Chop {
		t.Fatalf("the colonist didn't go for the tree, it's %v", colonistState(s, colonist))
	}

	site := groundAway(t, s, colonist, 3, tree)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	turned := false
	if !live(t, s, 20*data.TPS, func() bool {
		turned = turned || (isTree(s, tree) && colonistState(s, colonist) != inputsources.Colonist_Chop)
		return !isTree(s, tree)
	}) {
		t.Fatalf("the colonist didn't fell the tree, it's %v", colonistState(s, colonist))
	}
	if turned {
		t.Fatal("the colonist turned to the blueprint before felling the tree")
	}
	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatal("the colonist didn't build the blueprint once it had felled the tree")
	}
}

func TestAColonistDeliversWhatItGotWhenSomeoneTookPartOfWhatItWasFetching(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 10, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	s.Chunks.SetGroundStack(tile, groundWood(wallCost(t)))
	site := groundAway(t, s, colonist, 5, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	live(t, s, 2, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Fetch {
		t.Fatalf("the colonist didn't go to fetch, it's %v", colonistState(s, colonist))
	}

	s.Chunks.SetGroundStack(tile, groundWood(wallCost(t)-1))
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) > 0 }) {
		t.Fatal("the colonist didn't pick up what was left")
	}
	live(t, s, 1, func() bool { return false })
	job := colonistJob(s, colonist)
	if want := wallCost(t) - 1; job.State != inputsources.Colonist_Fetch || len(job.Stops) != 1 || job.Stops[0].Action != inputsources.Stop_Deliver || job.Stops[0].Count != want {
		t.Fatalf("the colonist is on %v, want delivering the %d it got", job.Stops, want)
	}
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) == 0 }) {
		t.Fatal("the colonist didn't deliver what it got")
	}

	other := storageSpot(t, s, colonist, 6, 1, 1, tile, site)
	markStorage(t, s, false, other)
	s.Chunks.SetGroundStack(other, groundWood(1))
	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the colonist didn't fetch the rest and build, it's %v", colonistState(s, colonist))
	}
}

func TestWhatWasFetchedForASiteThatsGoneIsPutBack(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 4, 1, 1)
	markStorage(t, s, false, tile)
	all := woodOntoStorage(t, s, colonist, tile)
	site := groundAway(t, s, colonist, 12, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) > 0 }) {
		t.Fatalf("the colonist didn't fetch, it's %v", colonistState(s, colonist))
	}

	if !s.Chunks.RemoveSite(site) {
		t.Fatal("couldn't remove the site")
	}
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) == 0 }) {
		t.Fatalf("the colonist kept what it fetched for a site that's gone, it's %v", colonistState(s, colonist))
	}
	if _, n := onGround(s, tile); n != all {
		t.Fatalf("storage holds %d wood, want all %d back", n, all)
	}
}

func TestAColonistHaulsOffASiteOntoStorageInOneGo(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 8, 1, 1)
	markStorage(t, s, false, tile)
	site := groundAway(t, s, colonist, 5, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	s.Chunks.SetGroundStack(site, groundTorches(3))

	live(t, s, 2, func() bool { return false })
	want := []inputsources.JobStop{
		{Action: inputsources.Stop_PickUp, Tile: site, Kind: data.Item_Torch, Count: 3},
		{Action: inputsources.Stop_PutDown, Tile: tile, Kind: data.Item_Torch, Count: 3},
	}
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Haul || !slices.Equal(job.Stops, want) {
		t.Fatalf("the colonist is on %v, want %v", job.Stops, want)
	}
	if !live(t, s, 20*data.TPS, func() bool { kind, n := onGround(s, tile); return kind == data.Item_Torch && n == 3 }) {
		t.Fatal("the torches weren't hauled onto storage")
	}
}

func TestTwoColonistsOnlyStoreWhatThereIsRoomFor(t *testing.T) {
	s, colony, a := colonistAtWork(t)
	b := anotherColonist(t, s, colony, a)
	tile := storageSpot(t, s, a, 6, 1, 1)
	markStorage(t, s, false, tile)
	s.Chunks.SetGroundStack(tile, groundWood(97))

	live(t, s, 2, func() bool { return false })
	ja, jb := colonistJob(s, a), colonistJob(s, b)
	if storing := (ja.State == inputsources.Colonist_Store) != (jb.State == inputsources.Colonist_Store); !storing {
		t.Fatalf("the colonists are %v and %v, want just one storing the 3 there's room for", ja.Stops, jb.Stops)
	}
	if !live(t, s, 20*data.TPS, func() bool { _, n := onGround(s, tile); return n == 100 }) {
		t.Fatal("storage wasn't filled up")
	}
}

func TestWhatsStoredGoesWithWhatsOnItsWayToStorage(t *testing.T) {
	s, colony, a := colonistAtWork(t)
	near := storageSpot(t, s, a, 4, 1, 1)
	far := storageSpot(t, s, a, 12, 1, 1, near)
	markStorage(t, s, false, near, far)
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, a); job.State != inputsources.Colonist_Store || job.Stops[0].Tile != near {
		t.Fatalf("the colonist is on %v, want storing on the closer storage", job.Stops)
	}

	b := anotherColonist(t, s, colony, a)
	testutil.SetPos(t, s.ECS, b, utils.TilePosToWorldPos(far))
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, b); job.State != inputsources.Colonist_Store || job.Stops[0].Tile != near {
		t.Fatalf("the other colonist is on %v, want storing where the first is", job.Stops)
	}
}

func TestAColonistFetchesFromSeveralStorageTilesInOneGo(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	a := storageSpot(t, s, colonist, 6, 2, 1)
	b := utils.Vec2i{X: a.X + 1, Y: a.Y}
	markStorage(t, s, false, a, b)
	withoutWood(t, s, colonist)
	s.Chunks.SetGroundStack(a, groundWood(1))
	s.Chunks.SetGroundStack(b, groundWood(wallCost(t)-1))
	site := groundAway(t, s, colonist, 10, a, b)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)

	live(t, s, 1, func() bool { return false })
	job := colonistJob(s, colonist)
	if job.State != inputsources.Colonist_Fetch || len(job.Stops) != 3 {
		t.Fatalf("the colonist is on %v, want picking up from both tiles, then delivering", job.Stops)
	}
	if !live(t, s, 20*data.TPS, func() bool { return len(colonistJob(s, colonist).Stops) == 2 }) {
		t.Fatal("the colonist didn't pick up from the first tile")
	}
	if last := colonistJob(s, colonist).Stops[1]; last.Count != wallCost(t) {
		t.Fatalf("after the first tile the colonist is to deliver %d, want all %d", last.Count, wallCost(t))
	}
	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the colonist didn't build the site, it's %v", colonistState(s, colonist))
	}
}

func TestAColonistDoesntFetchWhatItCantCarry(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)
	withoutWood(t, s, colonist)
	s.Chunks.SetGroundStack(tile, groundWood(10))
	torches, _ := s.ECS.SpawnItemStack(data.Item_Torch, data.ColonistMaxCarryWeight-s.ECS.CarriedWeight(colonist)-1, utils.Vec2f{})
	if err := s.ECS.PutInInventory(colonist, 0, torches); err != nil {
		t.Fatal(err)
	}
	site := groundAway(t, s, colonist, 8, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)

	live(t, s, 3*data.TPS, func() bool { return false })
	if state := colonistState(s, colonist); state == inputsources.Colonist_Fetch || woodCarried(s, colonist) != 0 {
		t.Fatalf("the colonist is %v carrying %d wood, with no room for all the site needs", state, woodCarried(s, colonist))
	}
}

func TestAColonistOnlyHaulsOntoStorageWhatThereIsRoomFor(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 8, 1, 1)
	markStorage(t, s, false, tile)
	s.Chunks.SetGroundStack(tile, groundTorches(98))
	site := groundAway(t, s, colonist, 5, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	s.Chunks.SetGroundStack(site, groundTorches(3))

	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Haul || len(job.Stops) != 2 || job.Stops[0].Count != 2 || job.Stops[1].Count != 2 {
		t.Fatalf("the colonist is on %v, want hauling the 2 there's room for", job.Stops)
	}
	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the site wasn't cleared and built, the colonist is %v", colonistState(s, colonist))
	}
	if _, n := onGround(s, tile); n != 100 {
		t.Fatalf("storage holds %d torches, want it filled to 100", n)
	}
}

func TestAStackAnotherColonistIsPickingUpAllOfIsLeftToIt(t *testing.T) {
	s, colony, a := colonistAtWork(t)
	withoutWood(t, s, a)
	lying := groundAway(t, s, a, 5)
	harvest(s, lying)
	looseWood(t, s, lying, 5)
	live(t, s, 1, func() bool { return false })

	b := anotherColonist(t, s, colony, a)
	withoutWood(t, s, b)
	live(t, s, 1, func() bool { return false })
	if state := colonistState(s, b); state != inputsources.Colonist_Idle {
		t.Fatalf("the other colonist is %v, with all of the stack being picked up", state)
	}
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, a) == 5 }) {
		t.Fatalf("the colonists carry %d and %d, want all 5 with the first", woodCarried(s, a), woodCarried(s, b))
	}
}

func TestOnlyOneColonistBuildsOrClearsASite(t *testing.T) {
	s, colony, a := colonistAtWork(t)
	tile := storageSpot(t, s, a, 8, 1, 1)
	markStorage(t, s, false, tile)
	cleared := groundAway(t, s, a, 5, tile)
	supplied := groundAway(t, s, a, 5, tile, cleared)
	s.Chunks.PlaceSite(cleared, data.Blueprint_WoodWall)
	s.Chunks.SetGroundStack(cleared, groundTorches(3))
	s.Chunks.PlaceSite(supplied, data.Blueprint_WoodWall)
	s.Chunks.DeliverToSite(supplied, data.Item_Wood, wallCost(t))
	s.Chunks.DeliverToSite(cleared, data.Item_Wood, wallCost(t))
	b := anotherColonist(t, s, colony, a)
	c := anotherColonist(t, s, colony, a)

	live(t, s, 1, func() bool { return false })
	states := map[inputsources.ColonistState]int{}
	for _, e := range []common.EntityId{a, b, c} {
		states[colonistState(s, e)]++
	}
	if states[inputsources.Colonist_Haul] != 1 || states[inputsources.Colonist_Construct] != 1 {
		t.Fatalf("the colonists are %v, want one clearing and one building", states)
	}
}

func TestAColonistOnlyPutsAwayWhatSitesPlacedOnItsWayDontNeed(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 10, 1, 1)
	markStorage(t, s, false, tile)
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Store || job.Stops[0].Count != data.ColonistStartingWood {
		t.Fatalf("the colonist is on %v, want putting away all its wood", job.Stops)
	}

	site := groundAway(t, s, colonist, 20, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	if !live(t, s, 20*data.TPS, func() bool { _, n := onGround(s, tile); return n > 0 }) {
		t.Fatal("the colonist didn't put anything away")
	}
	if _, n := onGround(s, tile); n != data.ColonistStartingWood-wallCost(t) {
		t.Fatalf("the colonist put away %d wood, want all but the %d the new site needs", n, wallCost(t))
	}
}

func TestAColonistsDebugLinesShowItsJobAndWhatItCarries(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 8, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	site := groundAway(t, s, colonist, 5, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	live(t, s, 1, func() bool { return false })

	lines, ok := inputsources.ColonistDebugLines(colonist, s.ECS)
	want := []string{
		fmt.Sprintf("Colonist %d: Fetch", colonist),
		fmt.Sprintf("> Pick up %d Wood at %d,%d", wallCost(t), tile.X, tile.Y),
		fmt.Sprintf("  Deliver %d Wood at %d,%d", wallCost(t), site.X, site.Y),
	}
	if !ok || len(lines) < len(want)+2 || !slices.Equal(lines[:len(want)], want) {
		t.Fatalf("the colonist's debug lines are %q, want them to start %q", lines, want)
	}
	if carrying := fmt.Sprintf("Carrying %d/%d", s.ECS.CarriedWeight(colonist), data.ColonistMaxCarryWeight); !slices.Contains(lines, carrying) || !slices.Contains(lines, "  Axe") {
		t.Fatalf("the colonist's debug lines %q don't show it carrying its axe, at %q", lines, carrying)
	}
	if _, ok := inputsources.ColonistDebugLines(s.Player, s.ECS); ok {
		t.Fatal("the player has a colonist's debug lines")
	}
}
