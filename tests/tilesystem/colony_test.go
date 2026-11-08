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

func spawnColony(t *testing.T, s *testutil.Session) common.EntityId {
	t.Helper()
	colony, err := templates.SpawnColonyCenter(s.ECS)
	if err != nil {
		t.Fatal(err)
	}
	s.Tick(1)
	return colony
}

func TestTheColonyIsTheThreeByThreeChunksAroundTheMiddleOfTheOriginChunk(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	colony := spawnColony(t, s)

	if tile := utils.WorldPosToTilePos(testutil.WorldPos(t, s.ECS, colony)); tile != (utils.Vec2i{X: 64, Y: 64}) {
		t.Fatalf("the colony center is on %v, want 64, 64", tile)
	}
	first, last, err := s.ECS.ColonyBounds(colony)
	if err != nil {
		t.Fatal(err)
	}
	if first != (utils.Vec2i{X: -data.ChunkSize, Y: -data.ChunkSize}) || last != (utils.Vec2i{X: 2*data.ChunkSize - 1, Y: 2*data.ChunkSize - 1}) {
		t.Fatalf("the colony goes from %v to %v, want the chunks from -1, -1 to 1, 1", first, last)
	}
	for tile, in := range map[utils.Vec2i]bool{first: true, last: true, {X: first.X - 1, Y: 0}: false, {X: 0, Y: last.Y + 1}: false} {
		if s.ECS.InColony(colony, tile) != in {
			t.Errorf("%v is in the colony %v, want %v", tile, !in, in)
		}
	}
	if _, _, err := s.ECS.ColonyBounds(s.Player); err == nil {
		t.Fatal("the player has colony bounds")
	}
}

func TestTheColonyStaysInTheSimulationWhereverThePlayerIs(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	colony := spawnColony(t, s)
	first, last, _ := s.ECS.ColonyBounds(colony)

	far := utils.Vec2f{X: 10 * data.ChunkSize * data.TileSize}
	s.MovePlayer(far)
	for range 200 {
		s.Tick(1)
		if s.Chunks.IsActiveWorldPos(far) {
			break
		}
	}
	if !s.Chunks.IsActiveWorldPos(far) {
		t.Fatal("the chunks around the player's new place never became active")
	}
	for _, corner := range []utils.Vec2i{first, last} {
		if !s.Chunks.IsActiveWorldPos(utils.TilePosToWorldPos(corner)) {
			t.Fatalf("the colony's corner %v left the simulation when the player went away", corner)
		}
	}

	s.Commit()
	r := testutil.StartSession(t)
	if !r.ECS.Colonies.HasComponent(colony) || !r.Chunks.IsActiveWorldPos(utils.TilePosToWorldPos(first)) {
		t.Fatal("after restarting, the colony center is gone or its chunks aren't in the simulation")
	}
}

func TestSitesAreListedInTileOrderWithinTheirBounds(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	from := utils.WorldPosToTilePos(testutil.PlayerStart)
	near := func(tile utils.Vec2i) bool { return utils.Dist2(tile, from) <= 40*40 && s.Chunks.CanPlaceSite(tile) }
	placed := []utils.Vec2i{}
	for range 6 {
		tile := findTile(t, s, near)
		s.Chunks.PlaceSite(tile, data.Blueprint_WoodWall)
		placed = append(placed, tile)
	}
	slices.SortFunc(placed, func(a, b utils.Vec2i) int {
		if a.Y != b.Y {
			return a.Y - b.Y
		}
		return a.X - b.X
	})

	if got := s.Chunks.SitesIn(utils.Vec2i{X: from.X - 40, Y: from.Y - 40}, utils.Vec2i{X: from.X + 40, Y: from.Y + 40}); !slices.Equal(got, placed) {
		t.Fatalf("listed %v, want %v", got, placed)
	}
	if got := s.Chunks.SitesIn(placed[1], placed[1]); !slices.Equal(got, placed[1:2]) {
		t.Fatalf("within the bounds of one site, listed %v", got)
	}
}
