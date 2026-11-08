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
func around(tiles ...data.TileLayers) func(utils.Vec2i) (data.TileLayers, bool) {
	return func(t utils.Vec2i) (data.TileLayers, bool) {
		if t.X < -1 || t.X > 0 || t.Y < -1 || t.Y > 0 {
			return data.TileLayers{}, false
		}
		return tiles[(t.Y+1)*2+t.X+1], true
	}
}

func layered(ground, cover, wall data.TileEnum) data.TileLayers {
	return data.TileLayers{data.TileLayer_Ground: ground, data.TileLayer_Cover: cover, data.TileLayer_Wall: wall}
}

func TestACornerIsDrawnLayerByLayerFromTheTilesAroundIt(t *testing.T) {
	s, _, _ := colonistAtWork(t)
	none, dirt := data.Tile_None, data.Tile_Dirt
	grass, bare := layered(dirt, data.Tile_Grass, none), layered(dirt, none, none)
	wall := layered(dirt, none, data.Tile_WoodWall)
	floor, rock := layered(data.Tile_RockFloor, none, none), layered(data.Tile_RockFloor, none, data.Tile_Rock)
	for _, c := range []struct {
		name  string
		tiles []data.TileLayers
		want  []tilesystem.CornerPiece
	}{
		{"all grass", []data.TileLayers{grass, grass, grass, grass}, []tilesystem.CornerPiece{{Tile: dirt, Cell: 6}, {Tile: data.Tile_Grass, Cell: 6}}},
		{"bare dirt in the bottom-right", []data.TileLayers{grass, grass, grass, bare}, []tilesystem.CornerPiece{{Tile: dirt, Cell: 6}, {Tile: data.Tile_Grass, Cell: 7}}},
		{"bare dirt on top, a wall at the bottom-left", []data.TileLayers{bare, bare, wall, grass}, []tilesystem.CornerPiece{
			{Tile: dirt, Cell: 6}, {Tile: data.Tile_Grass, Cell: 13}, {Tile: data.Tile_WoodWall, Cell: 0},
		}},
		{"walls on a diagonal", []data.TileLayers{wall, grass, grass, wall}, []tilesystem.CornerPiece{
			{Tile: dirt, Cell: 6}, {Tile: data.Tile_Grass, Cell: 14}, {Tile: data.Tile_WoodWall, Cell: 4},
		}},
		{"water under rock floor", []data.TileLayers{floor, layered(data.Tile_WaterDeep, none, none), layered(data.Tile_WaterShallow, none, none), floor}, []tilesystem.CornerPiece{
			{Tile: data.Tile_WaterDeep, Cell: 8}, {Tile: data.Tile_WaterShallow, Cell: 0}, {Tile: data.Tile_RockFloor, Cell: 4},
		}},
		{"rock around a cave", []data.TileLayers{floor, rock, rock, rock}, []tilesystem.CornerPiece{{Tile: data.Tile_RockFloor, Cell: 6}, {Tile: data.Tile_Rock, Cell: 5}}},
	} {
		pieces, ok := s.Chunks.CornerPieces(utils.Vec2i{}, around(c.tiles...))
		if !ok || !slices.Equal(pieces, c.want) {
			t.Errorf("with %s the corner is drawn as %+v (%v), want %+v", c.name, pieces, ok, c.want)
		}
	}
}

func TestATileDrawnWholeIsDrawnOverTheLayersUnderItAndUnderTheRestOfItsOwn(t *testing.T) {
	t.Chdir(testutil.RepoRoot())
	path := filepath.Join(t.TempDir(), "tiles.json")
	if err := os.WriteFile(path, []byte(`[
		{"id": 1, "imageAsset": "tile_dirt", "layer": "ground", "maxHealth": -1, "passable": true, "drawOrder": 0},
		{"id": 2, "imageAsset": "tile_planks", "layer": "cover", "maxHealth": -1, "passable": true},
		{"id": 3, "imageAsset": "tile_grass", "layer": "cover", "maxHealth": -1, "passable": true, "drawOrder": 0}
	]`), 0o644); err != nil {
		t.Fatal(err)
	}
	cc := tilesystem.NewChunkContainer()
	if err := cc.GenerateTileAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t)); err != nil {
		t.Fatal(err)
	}

	whole, sheet := data.TileLayers{1, 0, 2}, data.TileLayers{1, 0, 3}
	pieces, ok := cc.CornerPieces(utils.Vec2i{}, around(whole, sheet, sheet, sheet))
	want := []tilesystem.CornerPiece{{Tile: 1, Cell: 6}, {Tile: 2, Cell: -1, Quarter: 0}, {Tile: 3, Cell: 5}}
	if !ok || !slices.Equal(pieces, want) {
		t.Fatalf("the corner is drawn as %+v (%v), want %+v", pieces, ok, want)
	}
}

func TestACornerIsntDrawnWhileATileAroundItIsntKnown(t *testing.T) {
	s, _, _ := colonistAtWork(t)
	grass := layered(data.Tile_Dirt, data.Tile_Grass, data.Tile_None)
	known := around(grass, grass, grass, grass)
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
		if err := os.WriteFile(path, []byte(`[{"id": 1, "imageAsset": "`+c.asset+`", "layer": "ground", "maxHealth": -1, "passable": true, "drawOrder": 0}]`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := tilesystem.NewChunkContainer().GenerateTileAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t)); (err == nil) != c.ok {
			t.Errorf("drawing %s on the dual grid gave %v", c.asset, err)
		}
	}
}
