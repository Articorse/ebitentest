package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/abilitysystem"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/ecs/systems/commonsystems"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/ecs/systems/movementsystem"
	"ebittest/ecs/systems/timersystem"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"testing"
)

// Like live, but with bodies pushing each other and kept out of walls, as in the game, and how many times each colonist
// turned right round
func liveColliding(t *testing.T, s *testutil.Session, ticks int, colonists []common.EntityId) map[common.EntityId]int {
	t.Helper()
	turns := map[common.EntityId]int{}
	last := map[common.EntityId]utils.Vec2f{}
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
		for _, e := range colonists {
			dir := utils.Vec2f{X: inputs[e].Analog1X, Y: inputs[e].Analog1Y}
			if before, ok := last[e]; ok && before.X*dir.X+before.Y*dir.Y < -0.3 {
				turns[e]++
			}
			last[e] = dir
		}
		if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
			t.Fatal(err)
		}
		if err := timersystem.Tick(s.ECS); err != nil {
			t.Fatal(err)
		}
		if err := movementsystem.Tick(s.ECS); err != nil {
			t.Fatal(err)
		}
		colliders := ecs.Colliders(&s.ECS.PhysicsColliders)
		grid, err := commonsystems.PopulateSpatialHashGrid(s.ECS, data.SpatialHashGridCellSize)
		if err != nil {
			t.Fatal(err)
		}
		s.ECS.TickState.CollisionGrid = grid
		near, err := commonsystems.GetSHGProximities(grid, s.ECS)
		if err != nil {
			t.Fatal(err)
		}
		pairs := collisionsystem.GetAABBCollisions(colliders, near, s.ECS)
		if _, err := collisionsystem.ResolvePhysicsCollisions(collisionsystem.GetCollisions(colliders, colliders, pairs, s.ECS), s.ECS); err != nil {
			t.Fatal(err)
		}
		tiles, err := s.Chunks.GetTilesWithPotentialPhysicsCollisions(s.ECS)
		if err != nil {
			t.Fatal(err)
		}
		tilePairs, err := tilesystem.GetAABBCollisions(colliders, tiles, s.ECS)
		if err != nil {
			t.Fatal(err)
		}
		hits, err := tilesystem.GetCollisions(colliders, tilePairs, s.ECS)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tilesystem.ResolveTileCollisions(hits, s.ECS); err != nil {
			t.Fatal(err)
		}
		if err := abilitysystem.Tick(s.ECS); err != nil {
			t.Fatal(err)
		}
		touchTerrain(t, s)
		s.Tick(1)
	}
	return turns
}

// Five colonists walling in where they stand, fetching the wood from storage outside, which once had one of them turning
// back and forth on the spot it had picked, after a wall built since hid its site from there
func TestColonistsWallingARoomInTogetherDontGetStuck(t *testing.T) {
	s, colony, a := colonistAtWork(t)
	colonists := []common.EntityId{a}
	for range 4 {
		colonists = append(colonists, anotherColonist(t, s, colony, a))
	}
	corner := storageSpot(t, s, a, 9, 3, 3)
	storage := []utils.Vec2i{}
	for dy := range 3 {
		for dx := range 3 {
			storage = append(storage, utils.Vec2i{X: corner.X + dx, Y: corner.Y + dy})
		}
	}
	markStorage(t, s, false, storage...)
	for _, e := range colonists {
		withoutWood(t, s, e)
	}
	s.Chunks.SetGroundStack(corner, groundWood(80))

	center := colonistTile(t, s, a)
	walls := []utils.Vec2i{}
	for dy := -5; dy <= 5; dy++ {
		for dx := -5; dx <= 5; dx++ {
			tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
			if max(abs(dx), abs(dy)) == 5 && s.Chunks.CanPlaceSite(tile) && s.Chunks.PlaceSite(tile, data.Blueprint_Planks) {
				walls = append(walls, tile)
			}
		}
	}

	turns := liveColliding(t, s, 90*data.TPS, colonists)
	for _, tile := range walls {
		if _, ok := s.Chunks.Site(tile); ok {
			t.Errorf("the wall on %v wasn't built", tile)
		}
	}
	for e, n := range turns {
		if n > data.TPS {
			t.Errorf("colonist %d turned right round %d times", e, n)
		}
	}
}

func TestAColonistWhoseSiteIsHiddenFromWhereItStandsFindsSomewhereElse(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	site := groundAway(t, s, colonist, 8)
	s.Chunks.PlaceSite(site, data.Blueprint_Planks)
	s.Chunks.DeliverToSite(site, data.Item_Wood, planksCost(t))
	live(t, s, 1, func() bool { return false })
	job := colonistJob(s, colonist)
	stand := job.Stand
	if !live(t, s, 20*data.TPS, func() bool { return colonistTile(t, s, colonist) == stand }) {
		t.Fatal("the colonist didn't get to where it was going to build from")
	}

	between := utils.TilesAlongSegment(utils.TilePosToWorldPos(stand), utils.TilePosToWorldPos(site))
	for _, tile := range between[1 : len(between)-1] {
		s.Chunks.PlaceTile(tile, data.Tile_Planks)
	}
	if _, sees := s.ECS.AimedTileInReach(utils.TilePosToWorldPos(stand), utils.TilePosToWorldPos(site), data.PlayerBuildRangeTiles); sees {
		t.Fatal("the walls didn't hide the site")
	}
	if !live(t, s, 20*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the colonist didn't build the site from somewhere else, it's %v standing on %v", colonistState(s, colonist), colonistTile(t, s, colonist))
	}
}

func TestAnIdleColonistStepsOffASite(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := colonistTile(t, s, colonist)
	s.ECS.MarkArea(data.Area_Home, tile, tile, false)
	if !s.Chunks.PlaceSite(tile, data.Blueprint_Planks) {
		t.Fatal("couldn't place a site under the colonist")
	}

	if !live(t, s, 5*data.TPS, func() bool { return colonistTile(t, s, colonist) != tile }) {
		t.Fatal("the idle colonist stayed on the site")
	}
	if state := colonistState(s, colonist); state != inputsources.Colonist_Idle {
		t.Fatalf("the colonist is %v, want idle, with the site off home", state)
	}
	if _, onSite := s.Chunks.Site(colonistTile(t, s, colonist)); onSite {
		t.Fatal("the colonist stepped onto another site")
	}
}

func TestAColonistDoesntStandOnASiteToWorkOnAnother(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 5)
	for dx := range 5 {
		s.Chunks.PlaceSite(utils.Vec2i{X: row.X + dx, Y: row.Y}, data.Blueprint_Planks)
	}
	target := utils.Vec2i{X: row.X + 3, Y: row.Y}
	s.Chunks.DeliverToSite(target, data.Item_Wood, planksCost(t))
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(row))

	live(t, s, 1, func() bool { return false })
	job := colonistJob(s, colonist)
	if job.State != inputsources.Colonist_Construct || len(job.Stops) != 1 || job.Stops[0].Tile != target {
		t.Fatalf("the colonist is on %v, want building %v", job.Stops, target)
	}
	if _, onSite := s.Chunks.Site(job.Stand); onSite {
		t.Fatalf("the colonist is going to stand on the site on %v", job.Stand)
	}
}

func TestTwoColonistsDontPickTheSameSpotToStandOn(t *testing.T) {
	s, colony, a := colonistAtWork(t)
	row := openRow(t, s, a, 3)
	spot := row
	for _, dx := range []int{1, 2} {
		tile := utils.Vec2i{X: row.X + dx, Y: row.Y}
		s.Chunks.PlaceSite(tile, data.Blueprint_Planks)
		s.Chunks.DeliverToSite(tile, data.Item_Wood, planksCost(t))
	}
	testutil.SetPos(t, s.ECS, a, utils.TilePosToWorldPos(spot))
	b := anotherColonist(t, s, colony, a)

	live(t, s, 1, func() bool { return false })
	ja, jb := colonistJob(s, a), colonistJob(s, b)
	if ja.State != inputsources.Colonist_Construct || jb.State != inputsources.Colonist_Construct {
		t.Fatalf("the colonists are %v and %v, want both building", ja.State, jb.State)
	}
	if ja.Stand == jb.Stand {
		t.Fatalf("both colonists are going to stand on %v", ja.Stand)
	}
}

func TestASiteWithOnlySitesAroundItIsStillBuiltFromOne(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	r := data.PlayerBuildRangeTiles
	corner := storageSpot(t, s, colonist, 8, 2*r+1, 2*r+1)
	center := utils.Vec2i{X: corner.X + r, Y: corner.Y + r}
	for dy := range 2*r + 1 {
		for dx := range 2*r + 1 {
			s.Chunks.PlaceSite(utils.Vec2i{X: corner.X + dx, Y: corner.Y + dy}, data.Blueprint_Planks)
		}
	}
	s.Chunks.DeliverToSite(center, data.Item_Wood, planksCost(t))

	if !live(t, s, 30*data.TPS, isBuilt(s, center)) {
		t.Fatalf("the colonist didn't build the site in the middle, it's %v", colonistState(s, colonist))
	}
}
