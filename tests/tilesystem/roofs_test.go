package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"ebittest/worldgen"
	"fmt"
	"slices"
	"testing"
)

func TestMountainsStayRoofedWhenDugIntoAndAfterARestart(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	rock := tileOf(t, s, data.Tile_Rock)
	grass := tileOf(t, s, data.Tile_Grass)

	if roof, ok := s.Chunks.RoofAt(rock); !ok || roof != data.Roof_Rock {
		t.Fatalf("rock is roofed with %v (%v), want a rock roof", roof, ok)
	}
	if roof, _ := s.Chunks.RoofAt(grass); roof != data.Roof_None {
		t.Fatalf("grass is roofed with %v, want none", roof)
	}

	if ok, _ := s.Chunks.DamageTile(rock, 1000); !ok {
		t.Fatal("the rock couldn't be mined")
	}
	if roof, _ := s.Chunks.RoofAt(rock); roof != data.Roof_Rock {
		t.Fatalf("mined rock is roofed with %v, want the mountain's roof to stay", roof)
	}

	s.Commit()
	r := testutil.StartSession(t)
	if roof, _ := r.Chunks.RoofAt(rock); roof != data.Roof_Rock {
		t.Fatalf("after restarting, the dug-out tile is roofed with %v, want a rock roof", roof)
	}
}

func TestRoofsAreRemembered(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	open := findTile(t, s, func(tile utils.Vec2i) bool {
		roof, _ := s.Chunks.RoofAt(tile)
		return roof == data.Roof_Rock && !s.Chunks.IsOpaqueTile(tile)
	})
	// Roofed tiles are dark until something lights them
	seeLitBy(s, utils.TilePosToWorldPos(open), func(utils.Vec2i) uint8 { return data.MaxLight })
	s.Commit()

	r := testutil.StartSession(t)
	if roof := r.Chunks.RecallRoof(open); roof != data.Roof_Rock {
		t.Fatalf("a seen tile under a roof is remembered with %v, want a rock roof", roof)
	}
	if roof := r.Chunks.RecallRoof(utils.Vec2i{X: open.X + 10000}); roof != data.Roof_None {
		t.Fatalf("a tile never seen is remembered with roof %v", roof)
	}
}

// A chunk that's never been saved is written for the first time when an entity moves into it, and must be written whole
func TestAChunkFirstSavedForAnEntityKeepsItsTreesAndRoofs(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	e := testutil.AddMovable(t, s.ECS, nearPlayer, true)
	testutil.SetPos(t, s.ECS, e, farAway)
	s.Tick(3)
	s.Commit()

	chunk := utils.WorldPosToChunkGridPos(farAway)
	tree, roofed := utils.Vec2i{}, utils.Vec2i{}
	foundTree, foundRoof := false, false
	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			t := utils.Vec2i{X: chunk.X*data.ChunkSize + x, Y: chunk.Y*data.ChunkSize + y}
			if !foundTree && worldgen.Feature(testutil.WorldSeed, t) == data.Feature_Tree {
				tree, foundTree = t, true
			}
			if !foundRoof && worldgen.Roof(testutil.WorldSeed, t) != data.Roof_None {
				roofed, foundRoof = t, true
			}
		}
	}
	if !foundTree || !foundRoof {
		t.Fatalf("the chunk at %v has a tree %v and a roof %v, want both to check", chunk, foundTree, foundRoof)
	}

	s.MovePlayer(farAway)
	s.Tick(5)
	if f, ok := s.Chunks.FeatureAt(tree); !ok || f != data.Feature_Tree {
		t.Fatalf("the tree at %v is %v (%v) after its chunk was first saved for an entity", tree, f, ok)
	}
	if roof, _ := s.Chunks.RoofAt(roofed); roof == data.Roof_None {
		t.Fatalf("the roof at %v is gone after its chunk was first saved for an entity", roofed)
	}
}

func TestTheFaceOfAMountainIsSeenByDay(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	viewer := outside(t, s)
	v := see(s, viewer)

	center := utils.WorldPosToTilePos(viewer)
	for dy := -data.PlayerVisionRangeTiles; dy <= data.PlayerVisionRangeTiles; dy++ {
		for dx := -data.PlayerVisionRangeTiles; dx <= data.PlayerVisionRangeTiles; dx++ {
			tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
			if currentTile(s, tile) == data.Tile_Rock && v.CanSee(tile) {
				if got, ok := s.Chunks.RecallTile(tile); !ok || got.Top() != data.Tile_Rock {
					t.Fatalf("rock at %v is seen but remembered as %v (%v)", tile, got, ok)
				}
				return
			}
		}
	}
	t.Fatal("no rock is seen from beside a mountain by day, so mountains show as black")
}

// The middle of 17 by 17 tiles of open ground with no wall within 10 tiles, so that nothing there holds a roof up
func roofYard(t *testing.T, s *testutil.Session, colonist common.EntityId) utils.Vec2i {
	t.Helper()
	corner := groundAwayWhere(t, s, colonist, 20, func(corner utils.Vec2i) bool {
		for _, tile := range rect(corner, at(corner, 16, 16)) {
			if !s.Chunks.CanPlaceSite(tile) {
				return false
			}
		}
		middle := at(corner, 8, 8)
		return !slices.ContainsFunc(rect(at(middle, -10, -10), at(middle, 10, 10)), func(tile utils.Vec2i) bool {
			layers, _ := s.Chunks.TileAt(tile)
			return layers[data.TileLayer_Wall] != data.Tile_None
		})
	})
	return at(corner, 8, 8)
}

func roofOver(s *testutil.Session, tile utils.Vec2i) data.RoofEnum {
	roof, _ := s.Chunks.RoofAt(tile)
	return roof
}

// A wall above the tile, with a roof over it, which holds up a roof over the tile
func heldUpBeside(t *testing.T, s *testutil.Session, tile utils.Vec2i) utils.Vec2i {
	t.Helper()
	support := at(tile, 0, -1)
	wall(t, s, support)
	roof(t, s, []utils.Vec2i{support})
	return support
}

func TestARoofIsOnlyHeldUpThroughRoofsJoiningItToAWallOrDoor(t *testing.T) {
	for _, c := range []struct {
		name string
		// Built in order before the tile's roof is asked about
		walls, doors, roofs []utils.Vec2i
		supported           bool
	}{
		{"over a wall", []utils.Vec2i{{}}, nil, nil, true},
		{"over a door", nil, []utils.Vec2i{{}}, nil, true},
		{"beside a roofed wall", []utils.Vec2i{{Y: -1}}, nil, []utils.Vec2i{{Y: -1}}, true},
		{"beside a wall with no roof over it", []utils.Vec2i{{Y: -1}}, nil, nil, false},
		{"by the corner of a roofed wall", []utils.Vec2i{{X: 1, Y: -1}}, nil, []utils.Vec2i{{X: 1, Y: -1}}, false},
		{"6 steps along a roof from a roofed wall", []utils.Vec2i{{Y: -6}}, nil, line(utils.Vec2i{Y: -6}, utils.Vec2i{Y: -1}), true},
		{"7 steps along a roof from a roofed wall", []utils.Vec2i{{Y: -7}}, nil, line(utils.Vec2i{Y: -7}, utils.Vec2i{Y: -1}), false},
		{"6 steps round a corner of roof from a roofed door", nil, []utils.Vec2i{{X: -3, Y: -3}},
			append(line(utils.Vec2i{X: -3, Y: -3}, utils.Vec2i{X: 0, Y: -3}), line(utils.Vec2i{Y: -2}, utils.Vec2i{Y: -1})...), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, _, colonist := colonistAtWork(t)
			middle := roofYard(t, s, colonist)
			offset := func(d []utils.Vec2i) []utils.Vec2i {
				tiles := []utils.Vec2i{}
				for _, v := range d {
					tiles = append(tiles, at(middle, v.X, v.Y))
				}
				return tiles
			}
			if len(c.walls) > 0 {
				wall(t, s, offset(c.walls)...)
			}
			for _, d := range offset(c.doors) {
				buildDoor(t, s, d, data.Faction_Colony)
			}
			for _, r := range offset(c.roofs) {
				s.ECS.MarkArea(data.Area_Roof, r, r, true)
				if _, done := s.Chunks.WorkOnRoof(r, data.RoofWork); !done {
					t.Fatalf("couldn't roof %v on the way", r)
				}
			}
			s.ECS.MarkArea(data.Area_Roof, middle, middle, true)
			if got := s.Chunks.RoofJob(middle); got != c.supported {
				t.Fatalf("there's a roof to build %v, want %v", got, c.supported)
			}
		})
	}
}

// From one tile to another along a row or column, both included
func line(from, to utils.Vec2i) []utils.Vec2i {
	step := utils.Vec2i{X: sign(to.X - from.X), Y: sign(to.Y - from.Y)}
	tiles := []utils.Vec2i{from}
	for t := from; t != to; {
		t = at(t, step.X, step.Y)
		tiles = append(tiles, t)
	}
	return tiles
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

func TestARoofBesideARockRoofIsHeldUp(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	beside := findTile(t, s, func(tile utils.Vec2i) bool {
		if roofOver(s, tile) != data.Roof_None || !s.Chunks.CanPlaceSite(tile) {
			return false
		}
		for _, n := range []utils.Vec2i{at(tile, -1, 0), at(tile, 1, 0), at(tile, 0, -1), at(tile, 0, 1)} {
			if roofOver(s, n) == data.Roof_Rock {
				return true
			}
		}
		return false
	})
	s.Chunks.SetArea(data.Area_Roof, beside, beside, true)
	if !s.Chunks.RoofJob(beside) {
		t.Fatal("a roof beside a rock roof isn't held up")
	}
}

func TestAColonistRoofsWhereItsHeldUpAndTakesItDownOnTheNoRoofArea(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := roofYard(t, s, colonist)
	support := at(middle, 0, -4)
	wall(t, s, support)
	far := at(middle, 0, 4)
	s.ECS.MarkArea(data.Area_Roof, support, middle, true)
	s.ECS.MarkArea(data.Area_Roof, far, far, true)

	if !live(t, s, 30*data.TPS, func() bool { return roofOver(s, middle) == data.Roof_Built }) {
		t.Fatalf("the colonist didn't roof the tile a wall holds up, it's %v", colonistState(s, colonist))
	}
	if s.Chunks.IsOpenToSky(middle) {
		t.Fatal("the roofed tile is open to the sky")
	}
	live(t, s, 3*data.TPS, func() bool { return false })
	if roofOver(s, far) != data.Roof_None {
		t.Fatal("the colonist roofed the tile nothing holds up")
	}
	if colonistState(s, colonist) == inputsources.Colonist_Roof {
		t.Fatal("the colonist is still roofing with nothing left to roof")
	}

	s.ECS.MarkArea(data.Area_NoRoof, middle, middle, true)
	if s.Chunks.InArea(data.Area_Roof, middle) {
		t.Fatal("the tile stayed on the roof area once it was put on the no-roof area")
	}
	if !live(t, s, 20*data.TPS, func() bool { return roofOver(s, middle) == data.Roof_None }) {
		t.Fatalf("the colonist didn't take the roof down, it's %v", colonistState(s, colonist))
	}
	live(t, s, 3*data.TPS, func() bool { return false })
	if roofOver(s, middle) != data.Roof_None {
		t.Fatal("the roof was built again on the no-roof area")
	}
}

func TestTwoColonistsDontRoofTheSameTile(t *testing.T) {
	s, colony, colonist := colonistAtWork(t)
	middle := roofYard(t, s, colonist)
	wall(t, s, at(middle, -1, 0), middle, at(middle, 1, 0))
	s.ECS.MarkArea(data.Area_Roof, at(middle, -1, 0), at(middle, 1, 0), true)
	other := anotherColonist(t, s, colony, colonist)
	live(t, s, data.TPS, func() bool { return false })
	a, b := colonistJob(s, colonist), colonistJob(s, other)
	if a.State != inputsources.Colonist_Roof || b.State != inputsources.Colonist_Roof {
		t.Fatalf("the colonists are %v and %v, want both roofing", a.State, b.State)
	}
	if a.Subject == b.Subject {
		t.Fatalf("both colonists are roofing %v", a.Subject)
	}
}

func TestOnlyBuiltRoofsAreTakenDown(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	rock := findTile(t, s, func(tile utils.Vec2i) bool { return roofOver(s, tile) == data.Roof_Rock })
	s.Chunks.SetArea(data.Area_NoRoof, rock, rock, true)
	if s.Chunks.RoofJob(rock) {
		t.Fatal("there's a rock roof to take down")
	}
	if worked, _ := s.Chunks.WorkOnRoof(rock, 1000); worked || roofOver(s, rock) != data.Roof_Rock {
		t.Fatal("a rock roof was worked on")
	}
}

func TestBuiltRoofsFallInWithTheWallHoldingThemUp(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := roofYard(t, s, colonist)
	support := at(middle, 0, -5)
	wall(t, s, support)
	strip := line(support, middle)
	roof(t, s, strip)
	if destroyed, _ := s.Chunks.DamageTile(support, 1000); !destroyed {
		t.Fatal("the wall couldn't be knocked down")
	}
	for _, tile := range strip {
		if roofOver(s, tile) != data.Roof_None {
			t.Fatalf("the roof over %v stayed up once the wall holding up the roof it's part of was knocked down", tile)
		}
	}
}

func TestABuiltRoofFallsInOnceNothingHoldsItUp(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := roofYard(t, s, colonist)
	support, door := heldUpBeside(t, s, middle), at(middle, 1, 0)
	buildDoor(t, s, door, data.Faction_Colony)
	roof(t, s, []utils.Vec2i{door, middle})

	if destroyed, _ := s.Chunks.DamageTile(support, 1000); !destroyed {
		t.Fatal("the wall couldn't be knocked down")
	}
	if roofOver(s, middle) != data.Roof_Built || roofOver(s, support) != data.Roof_Built {
		t.Fatal("the roof fell in while the door still held it up")
	}
	if destroyed, _ := s.Chunks.DamageFeature(door, 1000); !destroyed {
		t.Fatal("the door couldn't be knocked down")
	}
	for _, tile := range []utils.Vec2i{support, middle, door} {
		if roofOver(s, tile) != data.Roof_None {
			t.Fatalf("the roof over %v stayed up with nothing holding it up", tile)
		}
	}
}

func TestTakingDownARoofDropsWhatItHeldUp(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := roofYard(t, s, colonist)
	support := at(middle, 0, -3)
	wall(t, s, support)
	strip := line(support, at(middle, 0, 3))
	roof(t, s, strip)
	cut := at(middle, 0, -1)
	unroof(t, s, cut)
	for _, tile := range strip {
		if want := tile.Y < cut.Y; (roofOver(s, tile) == data.Roof_Built) != want {
			t.Fatalf("after taking down the roof over %v, the roof over %v is up %v, want %v", cut, tile, !want, want)
		}
	}
}

func TestWorkOnARoofIsSavedAndCarriedOn(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := roofYard(t, s, colonist)
	heldUpBeside(t, s, middle)
	s.ECS.MarkArea(data.Area_Roof, middle, middle, true)
	if worked, done := s.Chunks.WorkOnRoof(middle, data.RoofWork-1); !worked || done {
		t.Fatalf("working on the roof worked %v and finished it %v, want worked and not finished", worked, done)
	}
	if !slices.Contains(s.Chunks.RoofDebugLines(middle), fmt.Sprintf("Building %d/%d", data.RoofWork-1, data.RoofWork)) {
		t.Fatalf("the tooltip shows %v, without the work done", s.Chunks.RoofDebugLines(middle))
	}
	s.Commit()

	r := testutil.StartSession(t)
	if _, done := r.Chunks.WorkOnRoof(middle, 1); !done {
		t.Fatal("the roof wasn't finished with the work done before the restart")
	}
	if roofOver(r, middle) != data.Roof_Built {
		t.Fatal("the finished roof isn't built")
	}
}

func TestTakingARoofDownDoesntCarryOnFromBuildingIt(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := roofYard(t, s, colonist)
	heldUpBeside(t, s, middle)
	s.ECS.MarkArea(data.Area_Roof, middle, middle, true)
	s.Chunks.WorkOnRoof(middle, data.RoofWork)
	s.Chunks.WorkOnRoof(middle, data.RoofWork-1)
	s.ECS.MarkArea(data.Area_NoRoof, middle, middle, true)
	if _, done := s.Chunks.WorkOnRoof(middle, 1); done {
		t.Fatal("the roof came down with the work of building it")
	}
}

func TestRoofingWaitsOutTheConstructorsCooldown(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := roofYard(t, s, colonist)
	heldUpBeside(t, s, middle)
	s.ECS.MarkArea(data.Area_Roof, middle, middle, true)
	s.ECS.SetDisabled(colonist, true)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(middle, 1, 0)))
	for range data.RoofWork {
		s.ECS.WorkOnRoof(colonist, middle)
	}
	if roofOver(s, middle) != data.Roof_None {
		t.Fatal("the roof was built in one go, without waiting between strikes")
	}
}

func TestRoofingIsLitAndSeenAfresh(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := roofYard(t, s, colonist)
	heldUpBeside(t, s, middle)
	s.ECS.MarkArea(data.Area_Roof, middle, middle, true)
	before := s.Chunks.SightVersion()
	s.Chunks.WorkOnRoof(middle, data.RoofWork)
	if s.Chunks.SightVersion() == before {
		t.Fatal("building the roof didn't have sight and light worked out again")
	}
}
