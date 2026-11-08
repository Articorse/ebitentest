package tilesystem_test

import (
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func bareGrass(t *testing.T, s *testutil.Session) utils.Vec2i {
	t.Helper()
	return findTile(t, s, func(tile utils.Vec2i) bool {
		_, hasFeature := s.Chunks.FeatureAt(tile)
		return currentTile(s, tile) == data.Tile_Grass && !hasFeature
	})
}

func TestPlacedWoodenWallsBlockAndBreakIntoDirtAndWood(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tile := bareGrass(t, s)

	if !s.Chunks.CanPlaceTile(tile) || !s.Chunks.PlaceTile(tile, data.Tile_WoodWall) {
		t.Fatal("a wooden wall wasn't placed on bare grass")
	}
	if s.Chunks.CanPlaceTile(tile) || s.Chunks.PlaceTile(tile, data.Tile_WoodWall) {
		t.Fatal("a wooden wall was placed on a wooden wall")
	}

	s.Commit()
	r := testutil.StartSession(t)
	center := utils.TilePosToWorldPos(tile)
	if !r.Chunks.IsOpaqueTile(tile) || !r.Chunks.IsBlockedAt(center) {
		t.Fatal("after restarting, the wall doesn't block sight and movement")
	}
	if health, ok := r.Chunks.TileHealth(tile); !ok || health != 100 {
		t.Fatalf("after restarting, the wall has %d health (damageable %v), want 100", health, ok)
	}

	if !r.ECS.DamageTile(tile, 99) || lyingAround(r, data.Item_Wood, tile) != 0 {
		t.Fatal("the wall dropped something before they were destroyed")
	}
	if !r.ECS.DamageTile(tile, 1) {
		t.Fatal("the wall couldn't be destroyed")
	}
	if currentTile(r, tile) != data.Tile_Dirt || r.Chunks.IsBlockedAt(center) {
		t.Fatal("a destroyed wall didn't turn into dirt")
	}

	if kind, n := onGround(r, tile); kind != data.Item_Wood || n != 2 || lyingAround(r, data.Item_Wood, tile) != 2 {
		t.Fatalf("a destroyed wall left %d of %v on its tile, want the 2 wood it drops there", n, kind)
	}
}

func TestTilesArePlacedOnlyOnBareGround(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)
	rock := tileOf(t, s, data.Tile_Rock)

	for name, tile := range map[string]utils.Vec2i{"a tree": tree, "rock": rock} {
		if s.Chunks.CanPlaceTile(tile) || s.Chunks.PlaceTile(tile, data.Tile_WoodWall) {
			t.Errorf("a wooden wall was placed on %s", name)
		}
	}

	if !s.ECS.DamageFeature(tree, 1000) {
		t.Fatal("the tree couldn't be cut down")
	}
	if s.Chunks.CanPlaceTile(tree) || s.Chunks.PlaceTile(tree, data.Tile_WoodWall) {
		t.Error("a wooden wall was placed on a stump")
	}

	if s.Chunks.PlaceTile(bareGrass(t, s), 999) {
		t.Error("a tile that isn't in the atlas was placed")
	}
}
