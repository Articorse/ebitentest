package tilesystem_test

import (
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The tiles around the corner at 0,0, top-left, top-right, bottom-left and bottom-right
func around(tiles ...data.TileEnum) func(utils.Vec2i) (data.TileEnum, bool) {
	return func(t utils.Vec2i) (data.TileEnum, bool) {
		if t.X < -1 || t.X > 0 || t.Y < -1 || t.Y > 0 {
			return data.Tile_None, false
		}
		return tiles[(t.Y+1)*2+t.X+1], true
	}
}

func TestACornerIsDrawnFromTheTilesAroundIt(t *testing.T) {
	s, _, _ := colonistAtWork(t)
	grass, dirt, wall := data.Tile_Grass, data.Tile_Dirt, data.Tile_WoodWall
	for _, c := range []struct {
		name  string
		tiles []data.TileEnum
		want  []tilesystem.CornerPiece
	}{
		{"all grass", []data.TileEnum{grass, grass, grass, grass}, []tilesystem.CornerPiece{{Tile: grass, Cell: 6}}},
		{"dirt in the bottom-right, under grass", []data.TileEnum{grass, grass, grass, dirt}, []tilesystem.CornerPiece{{Tile: dirt, Cell: 13}, {Tile: grass, Cell: 7}}},
		{"dirt on top, a wall at the bottom-left", []data.TileEnum{dirt, dirt, wall, grass}, []tilesystem.CornerPiece{{Tile: dirt, Cell: 9}, {Tile: grass, Cell: 13}, {Tile: wall, Cell: 0}}},
		{"walls on a diagonal", []data.TileEnum{wall, grass, grass, wall}, []tilesystem.CornerPiece{{Tile: grass, Cell: 14}, {Tile: wall, Cell: 4}}},
		{"water under rock floor", []data.TileEnum{data.Tile_RockFloor, data.Tile_WaterDeep, data.Tile_WaterShallow, data.Tile_RockFloor}, []tilesystem.CornerPiece{
			{Tile: data.Tile_WaterDeep, Cell: 8}, {Tile: data.Tile_WaterShallow, Cell: 0}, {Tile: data.Tile_RockFloor, Cell: 4},
		}},
		{"a tile drawn whole under the rest", []data.TileEnum{data.Tile_None, grass, grass, grass}, []tilesystem.CornerPiece{{Tile: data.Tile_None, Cell: -1, Quarter: 0}, {Tile: grass, Cell: 5}}},
	} {
		pieces, ok := s.Chunks.CornerPieces(utils.Vec2i{}, around(c.tiles...))
		if !ok || !slices.Equal(pieces, c.want) {
			t.Errorf("with %s the corner is drawn as %+v (%v), want %+v", c.name, pieces, ok, c.want)
		}
	}
}

func TestACornerIsntDrawnWhileATileAroundItIsntKnown(t *testing.T) {
	s, _, _ := colonistAtWork(t)
	known := around(data.Tile_Grass, data.Tile_Grass, data.Tile_Grass, data.Tile_Grass)
	if _, ok := s.Chunks.CornerPieces(utils.Vec2i{X: 1}, known); ok {
		t.Fatal("a corner with tiles around it that aren't known was drawn")
	}
}

func TestOnlyATileWithADualGridSheetIsDrawnOnTheDualGrid(t *testing.T) {
	t.Chdir(testutil.RepoRoot())
	for _, c := range []struct {
		asset string
		ok    bool
	}{{"tile_wood_wall", true}, {"tile_planks", false}} {
		path := filepath.Join(t.TempDir(), "tiles.json")
		if err := os.WriteFile(path, []byte(`[{"id": 1, "imageAsset": "`+c.asset+`", "maxHealth": -1, "passable": true, "drawOrder": 0}]`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := tilesystem.NewChunkContainer().GenerateTileAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t)); (err == nil) != c.ok {
			t.Errorf("drawing %s on the dual grid gave %v", c.asset, err)
		}
	}
}
