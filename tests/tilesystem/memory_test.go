package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"ebittest/worldgen"
	"math"
	"testing"
)

func see(s *testutil.Session, viewer utils.Vec2f) *ecs.Vision {
	v := ecs.NewVisionComponent(data.PlayerVisionRadiusTiles)
	v.Update(utils.WorldPosToTilePos(viewer), s.Chunks.IsOpaqueTile)
	s.Chunks.RememberSeenTiles(viewer, v)
	return v
}

func TestOnlyTilesTheCircleShowsCompletelyAreRemembered(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	viewer := testutil.PlayerStart.Add(utils.Vec2f{X: 5, Y: -3})
	v := see(s, viewer)
	clearRadius, _ := v.CircleRadii()

	remembered := 0
	center := utils.WorldPosToTilePos(viewer)
	for dy := -data.PlayerVisionRadiusTiles; dy <= data.PlayerVisionRadiusTiles; dy++ {
		for dx := -data.PlayerVisionRadiusTiles; dx <= data.PlayerVisionRadiusTiles; dx++ {
			tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
			got, ok := s.Chunks.RecallTile(tile)

			pos := utils.TilePosToWorldPos(tile)
			farX := math.Abs(pos.X-viewer.X) + data.TileSize/2
			farY := math.Abs(pos.Y-viewer.Y) + data.TileSize/2
			want := v.CanSee(tile) && farX*farX+farY*farY <= clearRadius*clearRadius

			if ok != want {
				t.Fatalf("tile %v remembered %v, want %v", tile, ok, want)
			}
			if ok && got != worldgen.Tile(testutil.WorldSeed, tile) {
				t.Fatalf("tile %v remembered as %v, but it's %v", tile, got, worldgen.Tile(testutil.WorldSeed, tile))
			}
			if ok {
				remembered++
			}
		}
	}
	if remembered == 0 {
		t.Fatal("nothing was remembered")
	}
}

func TestMemorySurvivesARestart(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	see(s, testutil.PlayerStart)
	tile := utils.WorldPosToTilePos(testutil.PlayerStart)
	want, ok := s.Chunks.RecallTile(tile)
	if !ok {
		t.Fatal("the viewer's own tile isn't remembered")
	}
	s.Commit()

	r := testutil.StartSession(t)
	if got, ok := r.Chunks.RecallTile(tile); !ok || got != want {
		t.Fatalf("after restarting, tile %v remembered %v as %v, want %v", tile, ok, got, want)
	}
}

func TestRememberingDuringACommitIsSavedByTheNext(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	see(s, testutil.PlayerStart)
	before := rememberedAround(s, testutil.PlayerStart)

	s.StartCommit()
	moved := testutil.PlayerStart.Add(utils.Vec2f{X: 4 * data.TileSize, Y: 2 * data.TileSize})
	see(s, moved)
	for s.Chunks.IsCommitting() {
		s.Tick(1)
	}

	during := []utils.Vec2i{}
	for tile := range rememberedAround(s, testutil.PlayerStart) {
		if !before[tile] {
			during = append(during, tile)
		}
	}
	if len(during) == 0 {
		t.Fatal("moving remembered nothing new")
	}

	s.Commit()
	r := testutil.StartSession(t)
	for _, tile := range during {
		if _, ok := r.Chunks.RecallTile(tile); !ok {
			t.Fatalf("tile %v, remembered while a commit was in flight, wasn't saved by the next", tile)
		}
	}
}

func rememberedAround(s *testutil.Session, pos utils.Vec2f) map[utils.Vec2i]bool {
	const reach = 2 * data.PlayerVisionRadiusTiles
	center := utils.WorldPosToTilePos(pos)
	remembered := map[utils.Vec2i]bool{}
	for dy := -reach; dy <= reach; dy++ {
		for dx := -reach; dx <= reach; dx++ {
			tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
			if _, ok := s.Chunks.RecallTile(tile); ok {
				remembered[tile] = true
			}
		}
	}
	return remembered
}
