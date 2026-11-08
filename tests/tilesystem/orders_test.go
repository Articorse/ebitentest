package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

// The sites as one order, in the order given, as placing them in one go does
func placeOrder(t *testing.T, s *testutil.Session, tiles ...utils.Vec2i) {
	t.Helper()
	s.ECS.LastOrder++
	for i, tile := range tiles {
		if !s.Chunks.PlaceSite(tile, data.Blueprint_WoodWall) || !s.Chunks.SetSiteOrder(tile, ecs.SiteOrder{Id: s.ECS.LastOrder, Seq: i}) {
			t.Fatalf("couldn't place site %d of the order on %v", i, tile)
		}
	}
}

// With torches taking up all it can carry but room for so much wood
func roomForWood(t *testing.T, s *testutil.Session, colonist common.EntityId, wood int) {
	t.Helper()
	inv, _ := s.ECS.Inventories.Get(colonist)
	torches, _ := s.ECS.SpawnItemStack(data.Item_Torch, inv.MaxWeight()-s.ECS.CarriedWeight(colonist)-2*wood, utils.Vec2f{})
	if err := s.ECS.PutInInventory(colonist, inv.FirstFreeSlot(), torches); err != nil {
		t.Fatal(err)
	}
	if room := s.ECS.RoomFor(colonist, data.Item_Wood); room != wood {
		t.Fatalf("the colonist has room for %d wood, want %d", room, wood)
	}
}

func rowOf(start utils.Vec2i, n int) []utils.Vec2i {
	tiles := []utils.Vec2i{}
	for i := range n {
		tiles = append(tiles, utils.Vec2i{X: start.X + i, Y: start.Y})
	}
	return tiles
}

func deliveriesOf(stops []inputsources.JobStop) []utils.Vec2i {
	tiles := []utils.Vec2i{}
	for _, s := range stops {
		if s.Action == inputsources.Stop_Deliver {
			tiles = append(tiles, s.Tile)
		}
	}
	return tiles
}

func TestPlacingBlueprintsInOneGoMakesThemOneOrderInTheOrderPlaced(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 4)
	tiles := rowOf(row, 4)
	slices.Reverse(tiles)
	taken := tiles[1]
	s.Chunks.PlaceFeature(taken, data.Feature_Torch)
	before := s.ECS.LastOrder

	handleInput(t, s, s.Player, ecs.InputState{PlaceBlueprint: ecs.BlueprintPlacement{Kind: data.Blueprint_WoodWall, Tiles: tiles}})
	seq := 0
	for _, tile := range tiles {
		site, ok := s.Chunks.Site(tile)
		if tile == taken {
			if ok {
				t.Fatal("a site was placed on the torch")
			}
			continue
		}
		if !ok || site.Order != (ecs.SiteOrder{Id: before + 1, Seq: seq}) {
			t.Fatalf("the site on %v is in order %+v, want %d at %d", tile, site.Order, before+1, seq)
		}
		seq++
	}

	other := groundAway(t, s, colonist, 3, tiles...)
	handleInput(t, s, s.Player, ecs.InputState{PlaceBlueprint: ecs.BlueprintPlacement{Kind: data.Blueprint_WoodWall, Tiles: []utils.Vec2i{other}}})
	if site, _ := s.Chunks.Site(other); site.Order != (ecs.SiteOrder{Id: before + 2}) {
		t.Fatalf("the next site placed is in order %+v, want the next order", site.Order)
	}

	s.Commit()
	r := testutil.StartSession(t)
	if site, _ := r.Chunks.Site(other); site.Order != (ecs.SiteOrder{Id: before + 2}) || r.ECS.LastOrder != before+2 {
		t.Fatalf("after restarting the site is in order %+v and the last order is %d", site.Order, r.ECS.LastOrder)
	}
}

func TestAColonistFetchesForTheSitesOfAnOrderInTurnAsFarAsItCanCarry(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	roomForWood(t, s, colonist, 3*wallCost(t)+1)
	sites := rowOf(storageSpot(t, s, colonist, 9, 5, 1, tile), 5)
	placeOrder(t, s, sites...)

	live(t, s, 1, func() bool { return false })
	job := colonistJob(s, colonist)
	if job.State != inputsources.Colonist_Fetch || !slices.Equal(deliveriesOf(job.Stops), sites[:3]) {
		t.Fatalf("the colonist is on %v, want fetching for the first three sites of the order", job.Stops)
	}
	if job.Stops[0] != (inputsources.JobStop{Action: inputsources.Stop_PickUp, Tile: tile, Kind: data.Item_Wood, Count: 3 * wallCost(t)}) {
		t.Fatalf("the colonist is picking up %v, want all three need at once", job.Stops[0])
	}
	if !live(t, s, 60*data.TPS, func() bool {
		return !slices.ContainsFunc(sites[:3], func(t utils.Vec2i) bool { return !isBuilt(s, t)() })
	}) {
		t.Fatal("the colonist didn't build the three sites it fetched for")
	}
}

func TestAColonistCarryingWoodDeliversAlongAnOrder(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	withoutWood(t, s, colonist)
	wood, _ := s.ECS.SpawnItemStack(data.Item_Wood, 2*wallCost(t)+1, utils.Vec2f{})
	if err := s.ECS.PutInInventory(colonist, 0, wood); err != nil {
		t.Fatal(err)
	}
	sites := rowOf(storageSpot(t, s, colonist, 6, 4, 1), 4)
	placeOrder(t, s, sites...)

	live(t, s, 1, func() bool { return false })
	job := colonistJob(s, colonist)
	if job.State != inputsources.Colonist_Deliver || !slices.Equal(deliveriesOf(job.Stops), sites[:2]) {
		t.Fatalf("the colonist is on %v, want delivering to the first two sites", job.Stops)
	}
}

func TestAColonistDeliversPartOfWhatASiteNeedsWhenItCantCarryAll(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	roomForWood(t, s, colonist, wallCost(t)-1)
	site := groundAway(t, s, colonist, 9, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)

	live(t, s, 1, func() bool { return false })
	job := colonistJob(s, colonist)
	if want := wallCost(t) - 1; job.State != inputsources.Colonist_Fetch || job.Stops[len(job.Stops)-1].Count != want {
		t.Fatalf("the colonist is on %v, want fetching the %d it can carry", job.Stops, want)
	}
	if !live(t, s, 60*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the colonist didn't build the site over more than one trip, it's on %v", colonistJob(s, colonist).Stops)
	}
}

// Two sites about as far from the colonist, carrying enough for one, with the one further away first
func twoSitesAsFar(t *testing.T, s *testutil.Session, colonist common.EntityId) (further, closer utils.Vec2i) {
	t.Helper()
	withoutWood(t, s, colonist)
	wood, _ := s.ECS.SpawnItemStack(data.Item_Wood, wallCost(t), utils.Vec2f{})
	if err := s.ECS.PutInInventory(colonist, 0, wood); err != nil {
		t.Fatal(err)
	}
	return twoSitesAsFarFrom(t, s, colonist)
}

// Two sites about as far from the colonist, with the one further away first
func twoSitesAsFarFrom(t *testing.T, s *testutil.Session, colonist common.EntityId, not ...utils.Vec2i) (further, closer utils.Vec2i) {
	t.Helper()
	further = groundAway(t, s, colonist, 6, not...)
	closer = groundAway(t, s, colonist, 6, append(not, further)...)
	if self := colonistTile(t, s, colonist); utils.Dist2(further, self) < utils.Dist2(closer, self) {
		further, closer = closer, further
	}
	s.Chunks.PlaceSite(further, data.Blueprint_WoodWall)
	s.Chunks.PlaceSite(closer, data.Blueprint_WoodWall)
	return further, closer
}

func TestAnEarlierOrderComesFirst(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	earlier, later := twoSitesAsFar(t, s, colonist)
	s.Chunks.SetSiteOrder(earlier, ecs.SiteOrder{Id: 1})
	s.Chunks.SetSiteOrder(later, ecs.SiteOrder{Id: 2})

	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Deliver || job.Stops[0].Tile != earlier {
		t.Fatalf("the colonist is on %v, want delivering to the earlier order's site on %v though it's a little further", job.Stops, earlier)
	}
}

func TestAnOrdersEarlierSitesComeFirst(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	first, last := twoSitesAsFar(t, s, colonist)
	s.Chunks.SetSiteOrder(first, ecs.SiteOrder{Id: 1, Seq: 0})
	s.Chunks.SetSiteOrder(last, ecs.SiteOrder{Id: 1, Seq: data.ColonistLaterSiteMaxCost / data.ColonistLaterSiteCost})

	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Deliver || job.Stops[0].Tile != first {
		t.Fatalf("the colonist is on %v, want delivering to the order's first site on %v though it's a little further", job.Stops, first)
	}
}

func TestAColonistCarriesOnWithTheOrderItWasBuilding(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	roomForWood(t, s, colonist, wallCost(t))
	sites := rowOf(storageSpot(t, s, colonist, 10, 3, 1, tile), 3)
	placeOrder(t, s, sites...)
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Fetch || !slices.Equal(deliveriesOf(job.Stops), sites[:1]) {
		t.Fatalf("the colonist is on %v, want fetching for the order's first site", job.Stops)
	}

	// Ready to build, and closer to storage than the rest of the order
	ready := groundAway(t, s, colonist, 3, tile, sites[0], sites[1], sites[2])
	s.Chunks.PlaceSite(ready, data.Blueprint_WoodWall)
	s.Chunks.DeliverToSite(ready, data.Item_Wood, wallCost(t))
	s.ECS.MarkArea(data.Area_Home, ready, ready, false)
	if !live(t, s, 30*data.TPS, isBuilt(s, sites[0])) {
		t.Fatal("the colonist didn't build the order's first site")
	}
	s.ECS.MarkArea(data.Area_Home, ready, ready, true)
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Fetch || !slices.Equal(deliveriesOf(job.Stops), sites[1:2]) {
		t.Fatalf("after building the colonist is %v on %v, want fetching for the order's next site", job.State, job.Stops)
	}
}

func TestTwoColonistsTakeSeparateLoadsOfOneOrder(t *testing.T) {
	s, colony, a := colonistAtWork(t)
	b := anotherColonist(t, s, colony, a)
	tile := storageSpot(t, s, a, 6, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, a, tile)
	withoutWood(t, s, b)
	s.Chunks.SetGroundStack(tile, groundWood(50))
	roomForWood(t, s, a, 3*wallCost(t))
	roomForWood(t, s, b, 3*wallCost(t))
	sites := rowOf(storageSpot(t, s, a, 10, 6, 1, tile), 6)
	placeOrder(t, s, sites...)

	live(t, s, 1, func() bool { return false })
	ja, jb := colonistJob(s, a), colonistJob(s, b)
	if !slices.Equal(deliveriesOf(ja.Stops), sites[:3]) || !slices.Equal(deliveriesOf(jb.Stops), sites[3:]) {
		t.Fatalf("the colonists are on %v and %v, want the first three sites and the last three", ja.Stops, jb.Stops)
	}
	if !live(t, s, 60*data.TPS, func() bool {
		return !slices.ContainsFunc(sites, func(t utils.Vec2i) bool { return !isBuilt(s, t)() })
	}) {
		t.Fatal("the colonists didn't build the whole order")
	}
}

func TestAColonistThatCantCarryMoreDeliversWhatItHas(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	withoutWood(t, s, colonist)
	wood, _ := s.ECS.SpawnItemStack(data.Item_Wood, wallCost(t)-1, utils.Vec2f{})
	if err := s.ECS.PutInInventory(colonist, 0, wood); err != nil {
		t.Fatal(err)
	}
	roomForWood(t, s, colonist, 0)
	site := groundAway(t, s, colonist, 6)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)

	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Deliver || job.Stops[0].Count != wallCost(t)-1 {
		t.Fatalf("the colonist is on %v, want delivering the %d it has", job.Stops, wallCost(t)-1)
	}
	if !live(t, s, 20*data.TPS, func() bool { return woodCarried(s, colonist) == 0 }) {
		t.Fatal("the colonist didn't deliver what it had")
	}
}

func TestWhatAColonistGotOfALoadGoesToTheOrdersSitesInTurn(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	s.Chunks.SetGroundStack(tile, groundWood(3*wallCost(t)))
	sites := rowOf(storageSpot(t, s, colonist, 10, 3, 1, tile), 3)
	placeOrder(t, s, sites...)
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); !slices.Equal(deliveriesOf(job.Stops), sites) {
		t.Fatalf("the colonist is on %v, want fetching for all three sites", job.Stops)
	}

	got := wallCost(t) + 1
	s.Chunks.SetGroundStack(tile, groundWood(got))
	if !live(t, s, 20*data.TPS, func() bool { return colonistJob(s, colonist).Stops[0].Action == inputsources.Stop_Deliver }) {
		t.Fatal("the colonist didn't pick up what was left")
	}
	want := []inputsources.JobStop{
		{Action: inputsources.Stop_Deliver, Tile: sites[0], Kind: data.Item_Wood, Count: wallCost(t)},
		{Action: inputsources.Stop_Deliver, Tile: sites[1], Kind: data.Item_Wood, Count: got - wallCost(t)},
	}
	if job := colonistJob(s, colonist); !slices.Equal(job.Stops, want) {
		t.Fatalf("with %d picked up the colonist is on %v, want %v", got, job.Stops, want)
	}
}

func TestABuilderCarriesOnAlongItsOrder(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	withoutWood(t, s, colonist)
	sites := rowOf(storageSpot(t, s, colonist, 8, 3, 1), 3)
	placeOrder(t, s, sites...)
	for _, site := range sites {
		s.Chunks.DeliverToSite(site, data.Item_Wood, wallCost(t))
	}
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Construct || job.Stops[0].Tile != sites[0] {
		t.Fatalf("the colonist is on %v, want building the order's first site", job.Stops)
	}

	// Ready to build too, closer to where the colonist builds from, but not part of the order
	other := groundNear(t, s, colonistJob(s, colonist).Stand, 2)
	s.Chunks.PlaceSite(other, data.Blueprint_WoodWall)
	s.Chunks.DeliverToSite(other, data.Item_Wood, wallCost(t))
	s.ECS.MarkArea(data.Area_Home, other, other, false)
	if !live(t, s, 30*data.TPS, isBuilt(s, sites[0])) {
		t.Fatal("the colonist didn't build the order's first site")
	}
	s.ECS.MarkArea(data.Area_Home, other, other, true)
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Construct || job.Stops[0].Tile != sites[1] {
		t.Fatalf("the colonist is on %v, want building the order's next site", job.Stops)
	}
}

func TestAColonistFetchesForAnEarlierOrderFirst(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 4, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	roomForWood(t, s, colonist, wallCost(t))
	earlier, later := twoSitesAsFarFrom(t, s, colonist, tile)
	s.Chunks.SetSiteOrder(earlier, ecs.SiteOrder{Id: 1})
	s.Chunks.SetSiteOrder(later, ecs.SiteOrder{Id: 2})

	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Fetch || !slices.Equal(deliveriesOf(job.Stops), []utils.Vec2i{earlier}) {
		t.Fatalf("the colonist is on %v, want fetching for the earlier order's site on %v though it's a little further", job.Stops, earlier)
	}
}

// Which once had it give up the rest of the load after the first site, seeing the site it had just supplied as no
// longer wanting anything
func TestAColonistDeliversTheWholeLoadInOneJob(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	sites := rowOf(storageSpot(t, s, colonist, 9, 3, 1, tile), 3)
	placeOrder(t, s, sites...)
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Fetch || !slices.Equal(deliveriesOf(job.Stops), sites) {
		t.Fatalf("the colonist is on %v, want fetching for all three sites", job.Stops)
	}

	supplied := func() bool {
		def, _ := testutil.Blueprints(t).Def(data.Blueprint_WoodWall)
		return !slices.ContainsFunc(sites, func(tile utils.Vec2i) bool {
			site, _ := s.Chunks.Site(tile)
			return site.Needs(data.Item_Wood, def) > 0
		})
	}
	gaveUp := false
	if !live(t, s, 40*data.TPS, func() bool {
		gaveUp = gaveUp || (!supplied() && colonistState(s, colonist) != inputsources.Colonist_Fetch)
		return supplied()
	}) {
		t.Fatal("the colonist didn't supply all three sites")
	}
	if gaveUp {
		t.Fatal("the colonist stopped fetching before it had supplied all three sites")
	}
}

func TestASiteInALoadSuppliedBySomeoneElseIsLeftOut(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	sites := rowOf(storageSpot(t, s, colonist, 9, 3, 1, tile), 3)
	placeOrder(t, s, sites...)
	live(t, s, 1, func() bool { return false })

	s.Chunks.DeliverToSite(sites[1], data.Item_Wood, wallCost(t))
	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Fetch || !slices.Equal(deliveriesOf(job.Stops), []utils.Vec2i{sites[0], sites[2]}) {
		t.Fatalf("the colonist is on %v, want still fetching for the other two sites", job.Stops)
	}
}

func TestALoadTakesTheSitesEitherSideOfWhereItStartsInOrder(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	roomForWood(t, s, colonist, 3*wallCost(t)+1)
	sites := rowOf(storageSpot(t, s, colonist, 10, 7, 1, tile), 7)
	placeOrder(t, s, sites...)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(sites[3]))

	live(t, s, 1, func() bool { return false })
	if job := colonistJob(s, colonist); job.State != inputsources.Colonist_Fetch || !slices.Equal(deliveriesOf(job.Stops), sites[2:5]) {
		t.Fatalf("the colonist is on %v, want fetching for the site it's closest to and the one either side, in order", job.Stops)
	}
}
