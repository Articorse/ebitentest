package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/shapes"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"os"
	"path/filepath"
	"testing"
)

func TestTilesAreWalkedAsFastAsTheirAtlasAndFeaturesSay(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	shallow := bareGrass(t, s)
	if !s.Chunks.PlaceTile(shallow, data.Tile_WaterShallow) {
		t.Fatal("shallow water wasn't placed")
	}
	deep := bareGrass(t, s)
	if !s.Chunks.PlaceTile(deep, data.Tile_WaterDeep) {
		t.Fatal("deep water wasn't placed")
	}
	torch := placeTorch(t, s)
	grass := bareGrass(t, s)

	for _, c := range []struct {
		name string
		tile utils.Vec2i
		want int
	}{
		{"grass", grass, 100},
		{"shallow water", shallow, 70},
		{"deep water", deep, 0},
		{"rock", tileOf(t, s, data.Tile_Rock), 0},
		{"a tree", findLoneTree(t, s), 0},
		{"a torch", torch, 100},
		{"an unloaded tile", utils.Vec2i{X: 1 << 20}, 0},
	} {
		if got := s.Chunks.WalkSpeedPercent(c.tile); got != c.want {
			t.Errorf("%s is walked at %d%%, want %d%%", c.name, got, c.want)
		}
	}

	if !s.Chunks.PlaceTile(grass, data.Tile_WoodWall) || s.Chunks.WalkSpeedPercent(grass) != 0 {
		t.Fatal("a wooden wall placed on grass that was walked on can be walked on")
	}
	tree := findLoneTree(t, s)
	s.Chunks.WalkSpeedPercent(tree)
	s.ECS.DamageFeature(tree, 1000)
	if got := s.Chunks.WalkSpeedPercent(tree); got != 100 {
		t.Fatalf("a felled tree's stump is walked at %d%%, want 100%%", got)
	}
}

func TestDeepWaterStopsWalkersButNotProjectiles(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	deep := bareGrass(t, s)
	if !s.Chunks.PlaceTile(deep, data.Tile_WaterDeep) {
		t.Fatal("deep water wasn't placed")
	}
	if layer, _ := s.Chunks.TileLayer(deep); layer != ecs.Layer_Water || !s.Chunks.IsBlockedAt(utils.TilePosToWorldPos(deep)) {
		t.Fatalf("deep water is on layer %b and blocked %v, want the water layer and blocked", layer, s.Chunks.IsBlockedAt(utils.TilePosToWorldPos(deep)))
	}

	center := utils.TilePosToWorldPos(deep)
	edge := center.Add(utils.Vec2f{X: -data.TileSize/2 - 2})
	walker := addCircle(t, s.ECS, edge, 3, func(shape shapes.Shape) ecs.Component {
		return ecs.NewPhysicsColliderComponent(ecs.Layer_Player, ecs.Layer_Terrain|ecs.Layer_Water, ecs.Collider_Mob, shape)
	})
	resolveTerrain(t, s)
	if got := testutil.WorldPos(t, s.ECS, walker); got.X >= edge.X {
		t.Fatalf("a walker reaching into deep water from the left moved from %v to %v, want it pushed out", edge, got)
	}

	bullet := addProjectile(t, s.ECS, center, ecs.Layer_Terrain|ecs.Layer_Feature)
	touchTerrain(t, s)
	if !s.ECS.HasEntity(bullet) {
		t.Fatal("a projectile over deep water died on it")
	}
}

func TestTheTileAtlasRefusesBrokenWalkingSpeeds(t *testing.T) {
	for _, c := range []struct {
		name string
		tile string
	}{
		{"a slowdown of 100%", `{"id": 1, "imageAsset": "tile_dirt", "maxHealth": -1, "passable": true, "slowdownPercent": 100}`},
		{"a negative slowdown", `{"id": 1, "imageAsset": "tile_dirt", "maxHealth": -1, "passable": true, "slowdownPercent": -1}`},
		{"passable water", `{"id": 1, "imageAsset": "tile_dirt", "maxHealth": -1, "passable": true, "water": true}`},
	} {
		t.Chdir(testutil.RepoRoot())
		path := filepath.Join(t.TempDir(), "tiles.json")
		if err := os.WriteFile(path, []byte(`[`+c.tile+`]`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := tilesystem.NewChunkContainer().GenerateTileAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t)); err == nil {
			t.Errorf("a tile with %s was accepted", c.name)
		}
	}
}
