package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestWhatLiesOnASiteIsHauledToStorageAndTheSiteBuilt(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first := storageSpot(t, s, colonist, 10, 2, 1)
	last := utils.Vec2i{X: first.X + 1, Y: first.Y}
	markStorage(t, s, false, first, last)
	site := groundAway(t, s, colonist, 5, first, last)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	s.Chunks.SetGroundStack(site, groundTorches(2))

	if !live(t, s, 40*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the blueprint wasn't built with torches lying on it, the colonist is %v", colonistState(s, colonist))
	}
	live(t, s, 10*data.TPS, func() bool { return stored(s, data.Item_Torch, first, last) == 2 })
	if n := stored(s, data.Item_Torch, first, last); n != 2 {
		t.Fatalf("storage holds %d torches, want the 2 hauled off the site", n)
	}
}

func TestWithoutStorageWhatLiesOnASiteIsMovedOffItNearby(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	site := groundAway(t, s, colonist, 5)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	s.Chunks.SetGroundStack(site, groundTorches(2))

	if !live(t, s, 40*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the blueprint wasn't built with torches lying on it, the colonist is %v", colonistState(s, colonist))
	}
	if lyingAround(s, data.Item_Torch, site) != 2 {
		t.Fatal("the torches weren't moved off the site nearby")
	}
	for _, t2 := range s.Chunks.GroundStacksIn(utils.Vec2i{X: site.X - 8, Y: site.Y - 8}, utils.Vec2i{X: site.X + 8, Y: site.Y + 8}) {
		if _, onSite := s.Chunks.Site(t2); onSite {
			t.Fatalf("the torches were moved onto another site at %v", t2)
		}
	}
}

func TestAUniqueItemOnASiteIsMovedOffEvenWithStorage(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 10, 1, 1)
	markStorage(t, s, false, tile)
	site := groundAway(t, s, colonist, 5, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	axe, _ := colonistAxe(s, colonist)
	if ok, err := s.ECS.LayDown(axe, site); err != nil || !ok {
		t.Fatalf("couldn't lay the axe down on the site: %v", err)
	}

	if !live(t, s, 40*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the blueprint wasn't built with the axe lying on it, the colonist is %v", colonistState(s, colonist))
	}
	found := false
	for _, t2 := range s.Chunks.GroundStacksIn(utils.Vec2i{X: site.X - 4, Y: site.Y - 4}, utils.Vec2i{X: site.X + 4, Y: site.Y + 4}) {
		stack, _ := s.Chunks.GroundStack(t2)
		found = found || stack.Unique != nil && stack.Unique.Root == axe && t2 != site
	}
	if !found {
		t.Fatal("the axe wasn't moved off the site nearby")
	}
}

func TestWhatLiesOnASiteOffHomeIsLeftAlone(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	site := groundAway(t, s, colonist, 5)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	s.Chunks.SetGroundStack(site, groundTorches(2))
	s.ECS.MarkArea(data.Area_Home, site, site, false)

	live(t, s, 5*data.TPS, func() bool { return false })
	if kind, n := onGround(s, site); kind != data.Item_Torch || n != 2 {
		t.Fatal("torches were hauled off a site that isn't on home")
	}
}

func TestMovingOnTheGroundOnlyMovesWhatFits(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 3)
	a, b, c := row, utils.Vec2i{X: row.X + 1, Y: row.Y}, utils.Vec2i{X: row.X + 2, Y: row.Y}
	s.ECS.LayDownKind(data.Item_Wood, 30*kg, a)
	s.Chunks.SetGroundStack(b, groundWood(90*kg))
	s.Chunks.SetGroundStack(c, groundTorches(1))

	if n := s.ECS.MoveOnGround(a, c); n != 0 {
		t.Fatal("wood was moved onto torches")
	}
	if n := s.ECS.MoveOnGround(a, b); n != 10*kg {
		t.Fatalf("moved %d wood onto a stack of 90, want the 10 that fit", n)
	}
	if _, n := onGround(s, a); n != 20*kg {
		t.Fatalf("%d wood was left behind, want 20", n)
	}
	axe, _ := colonistAxe(s, colonist)
	empty := groundAway(t, s, colonist, 8, a, b, c)
	if ok, err := s.ECS.LayDown(axe, empty); err != nil || !ok {
		t.Fatal(err)
	}
	if n := s.ECS.MoveOnGround(empty, a); n != 0 {
		t.Fatal("the axe was moved onto wood")
	}
	testutil.RemoveScheduled(t, s.ECS)
}

func TestAColonistStopsHaulingOffASiteTakenOffHome(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	site := groundAway(t, s, colonist, 10)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)
	s.Chunks.SetGroundStack(site, groundTorches(2))
	if !live(t, s, 2*data.TPS, func() bool { return colonistState(s, colonist) == inputsources.Colonist_Haul }) {
		t.Fatalf("the colonist didn't go to haul, it's %v", colonistState(s, colonist))
	}

	s.ECS.MarkArea(data.Area_Home, site, site, false)
	live(t, s, 1, func() bool { return false })
	if state := colonistState(s, colonist); state == inputsources.Colonist_Haul {
		t.Fatal("the colonist is still hauling off a site taken off home")
	}
	live(t, s, 5*data.TPS, func() bool { return false })
	if kind, n := onGround(s, site); kind != data.Item_Torch || n != 2 {
		t.Fatal("the colonist hauled off a site taken off home")
	}
}
