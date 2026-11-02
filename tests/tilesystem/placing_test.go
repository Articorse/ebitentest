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

func TestPlacedPlanksAreAWallThatBreaksIntoDirtAndAPlank(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tile := bareGrass(t, s)

	if !s.Chunks.CanPlaceTile(tile) || !s.Chunks.PlaceTile(tile, data.Tile_Planks) {
		t.Fatal("planks weren't placed on bare grass")
	}
	if s.Chunks.CanPlaceTile(tile) || s.Chunks.PlaceTile(tile, data.Tile_Planks) {
		t.Fatal("planks were placed on planks")
	}

	s.Commit()
	r := testutil.StartSession(t)
	center := utils.TilePosToWorldPos(tile)
	if !r.Chunks.IsOpaqueTile(tile) || !r.Chunks.IsBlockedAt(center) {
		t.Fatal("after restarting, the planks don't block sight and movement")
	}
	if health, ok := r.Chunks.TileHealth(tile); !ok || health != 100 {
		t.Fatalf("after restarting, the planks have %d health (damageable %v), want 100", health, ok)
	}

	items := len(r.ECS.Items.GetEntities())
	if !r.ECS.DamageTile(tile, 99) || len(r.ECS.Items.GetEntities()) != items {
		t.Fatal("the planks dropped something before they were destroyed")
	}
	if !r.ECS.DamageTile(tile, 1) {
		t.Fatal("the planks couldn't be destroyed")
	}
	if currentTile(r, tile) != data.Tile_Dirt || r.Chunks.IsBlockedAt(center) {
		t.Fatal("destroyed planks didn't turn into dirt")
	}

	dropped := len(r.ECS.Items.GetEntities()) - items
	if dropped != 1 {
		t.Fatalf("destroyed planks dropped %d items, want 1 plank", dropped)
	}
	for _, e := range r.ECS.Items.GetEntities() {
		if item, _ := r.ECS.Items.Get(e); item.Kind() == data.Item_Planks && item.Count() != 1 {
			t.Fatalf("destroyed planks dropped a stack of %d, want 1", item.Count())
		}
	}
}

func TestTilesArePlacedOnlyOnBareGround(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)
	rock := tileOf(t, s, data.Tile_Rock)

	for name, tile := range map[string]utils.Vec2i{"a tree": tree, "rock": rock} {
		if s.Chunks.CanPlaceTile(tile) || s.Chunks.PlaceTile(tile, data.Tile_Planks) {
			t.Errorf("planks were placed on %s", name)
		}
	}

	if !s.ECS.DamageFeature(tree, 1000) {
		t.Fatal("the tree couldn't be cut down")
	}
	if s.Chunks.CanPlaceTile(tree) || s.Chunks.PlaceTile(tree, data.Tile_Planks) {
		t.Error("planks were placed on a stump")
	}

	if s.Chunks.PlaceTile(bareGrass(t, s), 999) {
		t.Error("a tile that isn't in the atlas was placed")
	}
}
