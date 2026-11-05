package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/ecs/systems/movementsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"os"
	"path/filepath"
	"testing"
)

// A colonist standing where it can, near the colony center, and the buildable ground around it
func colonistAtWork(t *testing.T) (s *testutil.Session, colony, colonist common.EntityId) {
	t.Helper()
	testutil.TempWorld(t)
	s = testutil.StartSession(t)
	colony, colonist = spawnColonist(t, s)
	spawn, ok := s.Chunks.NewWorldSpawnPos(testutil.WorldPos(t, s.ECS, colony))
	if !ok {
		t.Fatal("there's nowhere to stand near the colony center")
	}
	testutil.SetPos(t, s.ECS, colonist, spawn)
	homeEverywhere(t, s, colony)
	return s, colony, colonist
}

func homeEverywhere(t *testing.T, s *testutil.Session, colony common.EntityId) {
	t.Helper()
	first, last, err := s.ECS.ColonyBounds(colony)
	if err != nil {
		t.Fatal(err)
	}
	s.ECS.MarkHome(first, last, true)
}

func colonistTile(t *testing.T, s *testutil.Session, colonist common.EntityId) utils.Vec2i {
	t.Helper()
	return utils.WorldPosToTilePos(testutil.WorldPos(t, s.ECS, colonist))
}

// Open ground the given distance from the colonist, give or take a tile, that a blueprint can go on
func groundAway(t *testing.T, s *testutil.Session, colonist common.EntityId, tiles int, not ...utils.Vec2i) utils.Vec2i {
	t.Helper()
	return groundAwayWhere(t, s, colonist, tiles, func(utils.Vec2i) bool { return true }, not...)
}

func groundAwayWhere(
	t *testing.T,
	s *testutil.Session,
	colonist common.EntityId,
	tiles int,
	also func(utils.Vec2i) bool,
	not ...utils.Vec2i,
) utils.Vec2i {
	t.Helper()
	from := colonistTile(t, s, colonist)
	r := tiles + 1
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			tile := utils.Vec2i{X: from.X + dx, Y: from.Y + dy}
			d := utils.Dist2(tile, from)
			if d < (tiles-1)*(tiles-1) || d > r*r || !s.Chunks.CanPlaceSite(tile) || !also(tile) {
				continue
			}
			clear := true
			for _, n := range not {
				clear = clear && utils.Dist2(tile, n) > 4*4
			}
			if clear {
				return tile
			}
		}
	}
	t.Fatalf("no ground to build on %d tiles from the colonist", tiles)
	return utils.Vec2i{}
}

// Runs every input source, the inputs and movement, which is all a colonist needs, until done or the ticks run out
func live(t *testing.T, s *testutil.Session, ticks int, done func() bool) bool {
	t.Helper()
	for range ticks {
		inputs := map[common.EntityId]ecs.InputState{}
		for e, in := range ecs.Query1(&s.ECS.Inputs, ecs.Without(&s.ECS.Disabled)) {
			source, err := s.ECS.InputSourceFunc(in.Type())
			if err != nil {
				t.Fatal(err)
			}
			if inputs[e], err = source(e, s.ECS.TickIdx, in.Params(), s.ECS); err != nil {
				t.Fatal(err)
			}
		}
		if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
			t.Fatal(err)
		}
		if err := movementsystem.Tick(s.ECS); err != nil {
			t.Fatal(err)
		}
		s.Tick(1)
		if done() {
			return true
		}
	}
	return false
}

func isBuilt(s *testutil.Session, tile utils.Vec2i) func() bool {
	return func() bool { return currentTile(s, tile) == data.Tile_Planks }
}

func colonistState(s *testutil.Session, colonist common.EntityId) inputsources.ColonistState {
	in, _ := s.ECS.Inputs.Get(colonist)
	return in.Params().(*inputsources.InputColonistParams).State
}

func woodCarried(s *testutil.Session, colonist common.EntityId) int {
	inv, _ := s.ECS.Inventories.Get(colonist)
	n := 0
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok {
			n += countOf(s, item)
		}
	}
	return n
}

func TestAColonistDeliversToAndBuildsTheClosestBlueprintFirst(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	near := groundAway(t, s, colonist, 8)
	far := groundAway(t, s, colonist, 16, near)
	s.Chunks.PlaceSite(far, data.Blueprint_Planks)
	s.Chunks.PlaceSite(near, data.Blueprint_Planks)

	if !live(t, s, 30*data.TPS, isBuilt(s, near)) {
		t.Fatalf("the colonist didn't build the closer blueprint, it's %v", colonistState(s, colonist))
	}
	if site, ok := s.Chunks.Site(far); !ok || site.Health != 1 {
		t.Fatal("the colonist worked on the further blueprint before the closer one was built")
	}
	if !live(t, s, 30*data.TPS, isBuilt(s, far)) {
		t.Fatal("the colonist didn't go on to build the further blueprint")
	}
	if woodCarried(s, colonist) != 20-2*planksCost(t) {
		t.Fatalf("the colonist has %d wood left, want what both blueprints cost taken from its 20", woodCarried(s, colonist))
	}
	live(t, s, 2*data.TPS, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatal("with nothing left to do, the colonist isn't idle")
	}
}

func TestBuildingComesBeforeDelivering(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	near := groundAway(t, s, colonist, 6)
	far := groundAway(t, s, colonist, 16, near)
	s.Chunks.PlaceSite(near, data.Blueprint_Planks)
	s.Chunks.PlaceSite(far, data.Blueprint_Planks)
	s.Chunks.DeliverToSite(far, data.Item_Wood, planksCost(t))

	if !live(t, s, 30*data.TPS, isBuilt(s, far)) {
		t.Fatal("the colonist didn't build the supplied site")
	}
	if site, ok := s.Chunks.Site(near); !ok || !site.IsBlueprint() {
		t.Fatal("the colonist delivered to the closer blueprint before building the supplied site")
	}
}

func TestAColonistLeavesBlueprintsItCantFullySupplyOrReachAlone(t *testing.T) {
	testutil.TempWorld(t)
	path := filepath.Join(t.TempDir(), "blueprints.json")
	json := `[{"id": 1, "name": "Pricey", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 21}], "work": 10},
		{"id": 2, "name": "Cheap", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 10}]`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	atlas, err := ecs.LoadBlueprintAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t))
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.StartSession(t)
	s.ECS.SetBlueprintAtlas(atlas)
	if err := s.Chunks.SetBlueprintAtlas(atlas); err != nil {
		t.Fatal(err)
	}
	colony, colonist := spawnColonist(t, s)
	spawn, _ := s.Chunks.NewWorldSpawnPos(testutil.WorldPos(t, s.ECS, colony))
	testutil.SetPos(t, s.ECS, colonist, spawn)
	homeEverywhere(t, s, colony)

	pricey := groundAway(t, s, colonist, 5)
	canBeWalledIn := func(tile utils.Vec2i) bool {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				n := utils.Vec2i{X: tile.X + dx, Y: tile.Y + dy}
				if n != tile && !s.Chunks.CanPlaceTile(n) && !s.Chunks.IsOpaqueTile(n) {
					return false
				}
			}
		}
		return true
	}
	walled := groundAwayWhere(t, s, colonist, 10, canBeWalledIn, pricey)
	s.Chunks.PlaceSite(pricey, 1)
	s.Chunks.PlaceSite(walled, 2)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if wall := (utils.Vec2i{X: walled.X + dx, Y: walled.Y + dy}); wall != walled {
				s.Chunks.DamageTile(wall, 1000)
				if !s.Chunks.PlaceTile(wall, data.Tile_Planks) && !s.Chunks.IsOpaqueTile(wall) {
					t.Fatalf("couldn't wall in the blueprint at %v", wall)
				}
			}
		}
	}

	start := colonistTile(t, s, colonist)
	live(t, s, 3*data.TPS, func() bool { return false })
	if site, ok := s.Chunks.Site(pricey); !ok || !site.IsBlueprint() {
		t.Fatal("the colonist delivered part of what it couldn't fully supply")
	}
	if site, ok := s.Chunks.Site(walled); !ok || !site.IsBlueprint() {
		t.Fatal("the colonist delivered to a blueprint walled in all around")
	}
	if colonistState(s, colonist) != inputsources.Colonist_Idle || colonistTile(t, s, colonist) != start {
		t.Fatal("with nothing it can do, the colonist didn't stay where it was")
	}

	cheap := groundAway(t, s, colonist, 8, pricey, walled)
	s.Chunks.PlaceSite(cheap, 2)
	if !live(t, s, 30*data.TPS, isBuilt(s, cheap)) {
		t.Fatal("the colonist didn't build a blueprint it could once one was placed")
	}
}

func TestAColonistLeavesBlueprintsOutsideTheColonyAlone(t *testing.T) {
	s, colony, colonist := colonistAtWork(t)
	_, last, _ := s.ECS.ColonyBounds(colony)
	s.MovePlayer(utils.TilePosToWorldPos(utils.Vec2i{X: last.X + 20, Y: 64}))
	outside := utils.Vec2i{}
	if !live(t, s, 200, func() bool {
		outside = utils.Vec2i{X: last.X + 3, Y: 64}
		for dy := range 40 {
			if t := (utils.Vec2i{X: outside.X, Y: 64 + dy}); s.Chunks.CanPlaceSite(t) {
				outside = t
				return true
			}
		}
		return false
	}) {
		t.Fatal("there's no ground to build on just outside the colony")
	}
	s.Chunks.PlaceSite(outside, data.Blueprint_Planks)

	live(t, s, 5*data.TPS, func() bool { return false })
	if site, ok := s.Chunks.Site(outside); !ok || !site.IsBlueprint() || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatal("the colonist went for a blueprint outside the colony")
	}
}

func TestAColonistDeliversAllThatsNeededAtOnce(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	site := groundAway(t, s, colonist, 8)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)

	if !live(t, s, 30*data.TPS, func() bool { state, _ := s.Chunks.Site(site); return !state.IsBlueprint() }) {
		t.Fatal("the colonist never delivered to the blueprint")
	}
	if state, _ := s.Chunks.Site(site); state.Delivered[0].Count != planksCost(t) {
		t.Fatalf("the colonist's first delivery was %d wood, want all %d at once", state.Delivered[0].Count, planksCost(t))
	}
}

func TestAColonistStandingOnASiteStepsOffToFinishIt(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	site := colonistTile(t, s, colonist)
	if !s.Chunks.PlaceSite(site, data.Blueprint_Planks) {
		t.Fatal("couldn't place a blueprint under the colonist")
	}
	s.Chunks.DeliverToSite(site, data.Item_Wood, planksCost(t))

	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatal("the colonist never built the site it was standing on")
	}
	if colonistTile(t, s, colonist) == site {
		t.Fatal("the colonist is inside the wall it built")
	}
}

func TestAColonistThatNoLongerHasEnoughGivesUp(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	site := groundAway(t, s, colonist, 12)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)
	live(t, s, 2, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Deliver {
		t.Fatal("the colonist didn't set out to deliver")
	}

	inv, _ := s.ECS.Inventories.Get(colonist)
	wood, _ := inv.Slot(0)
	for range 20 - planksCost(t) + 1 {
		if err := s.ECS.ConsumeItem(wood); err != nil {
			t.Fatal(err)
		}
	}
	live(t, s, 10*data.TPS, func() bool { return false })
	if state, ok := s.Chunks.Site(site); !ok || !state.IsBlueprint() || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatal("the colonist delivered part of what was needed after it no longer had all of it")
	}
}

func TestWhatAColonistIsDoingIsSaved(t *testing.T) {
	s, colony, colonist := colonistAtWork(t)
	site := groundAway(t, s, colonist, 12)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)
	live(t, s, 2, func() bool { return false })

	members, err := s.ECS.GetCompositionComponents(colonist)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)
	in, ok := y.Inputs.Get(colonist)
	if !ok {
		t.Fatal("the colonist's input wasn't saved")
	}
	p, ok := in.Params().(*inputsources.InputColonistParams)
	if !ok || p.Colony != colony || p.State != inputsources.Colonist_Deliver || p.Target != site {
		t.Fatalf("after loading, the colonist is %+v, want delivering to %v for colony %d", p, site, colony)
	}
}

func TestAColonistLeavesSitesOffHomeAlone(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	away := groundAway(t, s, colonist, 8)
	s.ECS.MarkHome(away, away, false)
	s.Chunks.PlaceSite(away, data.Blueprint_Planks)

	live(t, s, 3*data.TPS, func() bool { return false })
	if site, ok := s.Chunks.Site(away); !ok || !site.IsBlueprint() || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatal("the colonist went for a blueprint off home")
	}
	further := groundAway(t, s, colonist, 14, away)
	s.Chunks.PlaceSite(further, data.Blueprint_Planks)
	if !live(t, s, 30*data.TPS, isBuilt(s, further)) {
		t.Fatal("the colonist didn't build a blueprint on home further than one off it")
	}
	s.ECS.MarkHome(away, away, true)
	if !live(t, s, 30*data.TPS, isBuilt(s, away)) {
		t.Fatal("the colonist didn't build the blueprint once it was on home")
	}
}

func TestAColonistStopsWhenItsSiteIsTakenOffHome(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	site := groundAway(t, s, colonist, 12)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)
	live(t, s, 2, func() bool { return false })
	if colonistState(s, colonist) != inputsources.Colonist_Deliver {
		t.Fatal("the colonist didn't set out to deliver")
	}

	s.ECS.MarkHome(site, site, false)
	live(t, s, 10*data.TPS, func() bool { return false })
	if state, ok := s.Chunks.Site(site); !ok || !state.IsBlueprint() || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatal("the colonist carried on to a site taken off home")
	}
}

func TestAColonistGoesPastTheClosestBlueprintsWhenItCantGetToThem(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	// Three blueprints in a sealed room, closer than the one outside it
	const r = 3
	ringable := func(center utils.Vec2i) bool {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
				onRing := max(utils.AbsInt(dx), utils.AbsInt(dy)) == r
				if onRing && !s.Chunks.CanPlaceTile(tile) && !s.Chunks.IsOpaqueTile(tile) || !onRing && !s.Chunks.CanPlaceSite(tile) {
					return false
				}
			}
		}
		return true
	}
	room := groundAwayWhere(t, s, colonist, 8, ringable)
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if max(utils.AbsInt(dx), utils.AbsInt(dy)) == r {
				s.Chunks.PlaceTile(utils.Vec2i{X: room.X + dx, Y: room.Y + dy}, data.Tile_Planks)
			}
		}
	}
	sealed := []utils.Vec2i{room, {X: room.X - 1, Y: room.Y}, {X: room.X + 1, Y: room.Y}}
	for _, tile := range sealed {
		s.Chunks.PlaceSite(tile, data.Blueprint_Planks)
	}
	outside := groundAway(t, s, colonist, 16, room)
	s.Chunks.PlaceSite(outside, data.Blueprint_Planks)

	if !live(t, s, 30*data.TPS, isBuilt(s, outside)) {
		t.Fatal("the colonist didn't build the blueprint it could get to")
	}
	for _, tile := range sealed {
		if site, ok := s.Chunks.Site(tile); !ok || !site.IsBlueprint() {
			t.Fatalf("the colonist delivered to the blueprint at %v in a sealed room", tile)
		}
	}
}

func TestAColonistChoosesTheBlueprintClosestToWalkToOfTheClosestFew(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	// One 6 tiles east behind a wall across the way, closest as the crow flies but not to walk to, and one 8 tiles west
	const half = 8
	fits := func(self utils.Vec2i) bool {
		if s.Chunks.WalkSpeedPercent(self) <= 0 {
			return false
		}
		for x := self.X - 8; x <= self.X+6; x++ {
			if s.Chunks.WalkSpeedPercent(utils.Vec2i{X: x, Y: self.Y}) <= 0 && x != self.X+5 {
				return false
			}
		}
		for y := self.Y - half; y <= self.Y+half; y++ {
			if !s.Chunks.CanPlaceTile(utils.Vec2i{X: self.X + 5, Y: y}) {
				return false
			}
		}
		return s.Chunks.CanPlaceSite(utils.Vec2i{X: self.X + 6, Y: self.Y}) && s.Chunks.CanPlaceSite(utils.Vec2i{X: self.X - 8, Y: self.Y})
	}
	start := colonistTile(t, s, colonist)
	self, found := start, false
	for r := 0; r <= 40 && !found; r++ {
		for dy := -r; dy <= r && !found; dy++ {
			for dx := -r; dx <= r && !found; dx++ {
				self = utils.Vec2i{X: start.X + dx, Y: start.Y + dy}
				found = max(utils.AbsInt(dx), utils.AbsInt(dy)) == r && fits(self)
			}
		}
	}
	if !found {
		t.Fatal("there's no open ground for a wall near the colonist")
	}
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(self))
	for y := self.Y - half; y <= self.Y+half; y++ {
		s.Chunks.PlaceTile(utils.Vec2i{X: self.X + 5, Y: y}, data.Tile_Planks)
	}
	behind, open := utils.Vec2i{X: self.X + 6, Y: self.Y}, utils.Vec2i{X: self.X - 8, Y: self.Y}
	s.Chunks.PlaceSite(behind, data.Blueprint_Planks)
	s.Chunks.PlaceSite(open, data.Blueprint_Planks)

	if !live(t, s, 30*data.TPS, isBuilt(s, open)) {
		t.Fatal("the colonist didn't build the blueprint in the open")
	}
	if site, ok := s.Chunks.Site(behind); !ok || site.Health != 1 {
		t.Fatal("the colonist worked on the blueprint behind the wall first, though it's further to walk to")
	}
}
