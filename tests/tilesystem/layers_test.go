package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func layersAt(t *testing.T, s *testutil.Session, tile utils.Vec2i) data.TileLayers {
	t.Helper()
	tiles, ok := s.Chunks.TileAt(tile)
	if !ok {
		t.Fatalf("%v isn't loaded", tile)
	}
	return tiles
}

func TestAWallIsBuiltOverTheCoverAndLeavesTheGroundWhenDestroyed(t *testing.T) {
	for _, c := range []struct {
		name   string
		ground data.TileEnum
		find   func(*testing.T, *testutil.Session) utils.Vec2i
	}{
		{"on grass", data.Tile_Dirt, bareGrass},
		{"in a mined tunnel", data.Tile_RockFloor, func(t *testing.T, s *testutil.Session) utils.Vec2i {
			rock := tileOf(t, s, data.Tile_Rock)
			if mined, _ := s.Chunks.DamageTile(rock, 1000); !mined {
				t.Fatal("the rock couldn't be mined")
			}
			return rock
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			testutil.TempWorld(t)
			s := testutil.StartSession(t)
			tile := c.find(t, s)
			cover := layersAt(t, s, tile)[data.TileLayer_Cover]
			if !s.Chunks.PlaceSite(tile, data.Blueprint_WoodWall) {
				t.Fatal("the wall couldn't be placed")
			}
			s.Chunks.DeliverToSite(tile, data.Item_Wood, wallCost(t))
			if worked, _ := s.Chunks.BuildSite(tile, 1, false); !worked || layersAt(t, s, tile)[data.TileLayer_Cover] != cover {
				t.Fatal("the cover was cleared before the wall was finished")
			}

			if _, built := s.Chunks.BuildSite(tile, 1_000_000, true); !built {
				t.Fatal("the wall wasn't built")
			}
			want := data.TileLayers{data.TileLayer_Ground: c.ground, data.TileLayer_Wall: data.Tile_WoodWall}
			if got := layersAt(t, s, tile); got != want {
				t.Fatalf("the built wall's tile is %v, want %v", got, want)
			}

			if destroyed, _ := s.Chunks.DamageTile(tile, 1000); !destroyed {
				t.Fatal("the wall couldn't be destroyed")
			}
			if got := layersAt(t, s, tile); got != (data.TileLayers{data.TileLayer_Ground: c.ground}) {
				t.Fatalf("the destroyed wall left %v, want only the ground %v", got, c.ground)
			}
		})
	}
}

// A copy of the game's atlas in the world being tested, with a bridge, which is a foundation
func withABridge(t *testing.T) data.TileEnum {
	t.Helper()
	const bridge = data.TileEnum(100)
	var defs []map[string]any
	raw, err := os.ReadFile(testutil.TileAtlasPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &defs); err != nil {
		t.Fatal(err)
	}
	defs = append(defs, map[string]any{
		"id": bridge, "imageAsset": "tile_planks", "layer": "foundation", "maxHealth": 40, "passable": true, "buildable": true,
	})
	if raw, err = json.Marshal(defs); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testutil.TileAtlasPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return bridge
}

func TestAFoundationIsWalkedAndBuiltOnInsteadOfTheGroundUnderIt(t *testing.T) {
	testutil.TempWorld(t)
	bridge := withABridge(t)
	s := testutil.StartSession(t)
	water := tileOf(t, s, data.Tile_WaterShallow)
	if !s.Chunks.PlaceTile(water, bridge) {
		t.Fatal("the bridge couldn't be put over the water")
	}
	if speed := s.Chunks.WalkSpeedPercent(water); speed != 100 || !s.Chunks.CanPlaceSite(water) {
		t.Fatalf("the bridge is walked at %d%% and can be built on %v, want full speed and buildable", speed, s.Chunks.CanPlaceSite(water))
	}

	s.Commit()
	r := testutil.StartSession(t)
	if got := layersAt(t, r, water); got != (data.TileLayers{data.TileLayer_Ground: data.Tile_WaterShallow, data.TileLayer_Foundation: bridge}) {
		t.Fatalf("after restarting, the bridge's tile is %v", got)
	}

	if destroyed, _ := r.Chunks.DamageTile(water, 1000); !destroyed {
		t.Fatal("the bridge couldn't be destroyed")
	}
	if speed := r.Chunks.WalkSpeedPercent(water); speed != 70 || r.Chunks.CanPlaceSite(water) {
		t.Fatalf("without the bridge, the water is walked at %d%% and can be built on %v, want it slow and not buildable", speed, r.Chunks.CanPlaceSite(water))
	}
}

func TestDeepWaterUnderABridgeIsntWater(t *testing.T) {
	testutil.TempWorld(t)
	bridge := withABridge(t)
	s := testutil.StartSession(t)
	tile := bareGrass(t, s)
	// Laid in this order, since nothing can be put on deep water
	if dug, _ := s.Chunks.DamageTile(tile, 1000); !dug || !s.Chunks.PlaceTile(tile, bridge) || !s.Chunks.PlaceTile(tile, data.Tile_WaterDeep) {
		t.Fatal("couldn't put a bridge over deep water")
	}
	if layer, _ := s.Chunks.TileCollisionLayer(tile); layer != ecs.Layer_Floor || s.Chunks.WalkSpeedPercent(tile) != 100 {
		t.Fatalf("the bridge over deep water is on %v and walked at %d%%, want walked over", layer, s.Chunks.WalkSpeedPercent(tile))
	}
	if !s.Chunks.PlaceTile(tile, data.Tile_WoodWall) {
		t.Fatal("couldn't put a wall on the bridge")
	}
	if layer, _ := s.Chunks.TileCollisionLayer(tile); layer != ecs.Layer_Wall {
		t.Fatalf("the wall on the bridge is on %v, want a wall, which stops what flies over water", layer)
	}
}

func TestATileMustBeOnALayerThatCanBeWhatItIs(t *testing.T) {
	t.Chdir(testutil.RepoRoot())
	for _, c := range []struct {
		name, def string
		ok        bool
	}{
		{"buildable ground", `"layer": "ground", "buildable": true`, true},
		{"buildable foundation", `"layer": "foundation", "buildable": true`, true},
		{"on no layer", `"layer": "roof"`, false},
		{"buildable cover", `"layer": "cover", "buildable": true`, false},
		{"water that isn't ground", `"layer": "wall", "water": true`, false},
		{"ground that can be destroyed", `"layer": "ground", "maxHealth": 10`, false},
	} {
		path := filepath.Join(t.TempDir(), "tiles.json")
		def := `[{"id": 1, "imageAsset": "tile_dirt", "maxHealth": -1, "drawOrder": 0, ` + c.def + `}]`
		if err := os.WriteFile(path, []byte(def), 0o644); err != nil {
			t.Fatal(err)
		}
		cc := tilesystem.NewChunkContainer()
		t.Cleanup(cc.StopIOThread)
		if err := cc.GenerateTileAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t)); (err == nil) != c.ok {
			t.Errorf("a tile that's %s loaded with %v", c.name, err)
		}
	}
	t.Run("the empty tile", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "tiles.json")
		if err := os.WriteFile(path, []byte(`[{"id": 0, "imageAsset": "tile_dirt", "layer": "ground", "maxHealth": -1, "drawOrder": 0}]`), 0o644); err != nil {
			t.Fatal(err)
		}
		cc := tilesystem.NewChunkContainer()
		t.Cleanup(cc.StopIOThread)
		if err := cc.GenerateTileAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t)); err == nil {
			t.Error("a tile was defined as what an empty layer holds")
		}
	})
}

func TestACoverIsWalkedOverAtItsOwnSpeed(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tile := bareGrass(t, s)
	if dug, _ := s.Chunks.DamageTile(tile, 1000); !dug || !s.Chunks.PlaceTile(tile, data.Tile_WaterShallow) || !s.Chunks.PlaceTile(tile, data.Tile_Grass) {
		t.Fatal("couldn't put grass over shallow water")
	}
	if speed := s.Chunks.WalkSpeedPercent(tile); speed != 100 {
		t.Fatalf("grass over shallow water is walked at %d%%, want the grass's full speed", speed)
	}
}
