package tilesystem_test

import (
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"ebittest/worldgen"
	"testing"
)

func TestOpacityFollowsTheAtlasAndUnloadedChunksAreOpaque(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	center := utils.WorldPosToTilePos(testutil.PlayerStart)

	checked := map[bool]bool{}
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
			rock := worldgen.Tile(testutil.WorldSeed, tile) == data.Tile_Rock
			if got := s.Chunks.IsOpaqueTile(tile); got != rock {
				t.Fatalf("tile %v opaque %v, want %v", tile, got, rock)
			}
			checked[rock] = true
		}
	}
	if !checked[false] {
		t.Fatal("no open tile near the start to check")
	}

	if !s.Chunks.IsOpaqueTile(utils.Vec2i{X: 100000, Y: 100000}) {
		t.Fatal("a tile in an unloaded chunk isn't opaque")
	}
}
