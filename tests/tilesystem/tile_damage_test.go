package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"os"
	"path/filepath"
	"testing"
)

func tileOf(t *testing.T, s *testutil.Session, kind data.TileEnum) utils.Vec2i {
	t.Helper()
	return findTile(t, s, func(tile utils.Vec2i) bool { return currentTile(s, tile) == kind })
}

// What's there now, which can differ from what was generated
func currentTile(s *testutil.Session, tile utils.Vec2i) data.TileEnum {
	if _, ok := s.Chunks.TileHealth(tile); !ok && !s.Chunks.IsOpaqueTile(tile) {
		return data.Tile_Dirt
	}
	if s.Chunks.IsOpaqueTile(tile) {
		return data.Tile_Rock
	}
	return data.Tile_Grass
}

func TestDamagedTilesKeepTheirHealthAndBecomeDirt(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	grass := tileOf(t, s, data.Tile_Grass)
	rock := tileOf(t, s, data.Tile_Rock)

	if !s.Chunks.DamageTile(grass, 20) {
		t.Fatal("grass can't be damaged")
	}
	if health, _ := s.Chunks.TileHealth(grass); health != 30 {
		t.Fatalf("grass has %d health after 20 damage, want 30", health)
	}

	if !s.Chunks.DamageTile(rock, 500) {
		t.Fatal("rock can't be damaged")
	}
	if s.Chunks.IsOpaqueTile(rock) {
		t.Fatal("destroyed rock still blocks sight")
	}
	if _, ok := s.Chunks.TileHealth(rock); ok {
		t.Fatal("destroyed rock turned into something that can still be damaged, want dirt")
	}
	if s.Chunks.DamageTile(rock, 10) {
		t.Fatal("dirt took damage")
	}

	s.Commit()
	r := testutil.StartSession(t)
	if health, ok := r.Chunks.TileHealth(grass); !ok || health != 30 {
		t.Fatalf("after restarting, grass has %d health (damageable %v), want 30", health, ok)
	}
	if r.Chunks.IsOpaqueTile(rock) {
		t.Fatal("after restarting, destroyed rock is back")
	}
}

func addExplosion(t *testing.T, x *ecs.ECSContainer, pos utils.Vec2f, mask ecs.LayerMask, radii []float64, tiers ...int) common.EntityId {
	t.Helper()
	circles := []shapes.Shape{}
	for _, r := range radii {
		c, err := shapes.NewCircleShape(r, utils.Vec2f{})
		if err != nil {
			t.Fatal(err)
		}
		circles = append(circles, c)
	}
	return testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(pos, 1, 0),
		ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, mask, circles...),
		ecs.NewContactDamageComponent(-1, 0, false, true, tiers...))
}

func TestSingleTickExplosionDamagesEachTileOnceByItsRing(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	center := tileOf(t, s, data.Tile_Grass)
	next := utils.Vec2i{X: center.X + 1, Y: center.Y}
	if currentTile(s, next) != data.Tile_Grass {
		center = findTile(t, s, func(tile utils.Vec2i) bool {
			return currentTile(s, tile) == data.Tile_Grass && currentTile(s, utils.Vec2i{X: tile.X + 1, Y: tile.Y}) == data.Tile_Grass
		})
		next = utils.Vec2i{X: center.X + 1, Y: center.Y}
	}

	e := addExplosion(t, s.ECS, utils.TilePosToWorldPos(center), ecs.Layer_Terrain|ecs.Layer_Ground, []float64{4, 12}, 7, 3)
	touchTerrain(t, s)
	touchTerrain(t, s)

	if health, _ := s.Chunks.TileHealth(center); health != 50-7 {
		t.Fatalf("center tile has %d health, want %d: hit once, by the inner ring", health, 50-7)
	}
	if health, _ := s.Chunks.TileHealth(next); health != 50-3 {
		t.Fatalf("neighbouring tile has %d health, want %d: hit once, by the outer ring", health, 50-3)
	}
	if hurtbox, _ := s.ECS.HurtboxColliders.Get(e); hurtbox.Enabled() {
		t.Fatal("the single-tick hurtbox is still enabled")
	}
}

func TestSingleTickHurtboxThatHitsNothingStopsAfterItsFirstTick(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	e := addExplosion(t, s.ECS, testutil.PlayerStart, ecs.Layer_Enemy, []float64{5}, 1)

	touchTerrain(t, s)

	if hurtbox, _ := s.ECS.HurtboxColliders.Get(e); hurtbox.Enabled() {
		t.Fatal("the single-tick hurtbox is still enabled after its first tick")
	}
}

func TestProjectileWithoutGroundInItsMaskLeavesTheGroundAlone(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	grass := tileOf(t, s, data.Tile_Grass)

	e := addProjectile(t, s.ECS, utils.TilePosToWorldPos(grass), ecs.Layer_Enemy|ecs.Layer_Terrain)
	touchTerrain(t, s)

	if health, _ := s.Chunks.TileHealth(grass); health != 50 {
		t.Fatalf("grass under a projectile has %d health, want it untouched", health)
	}
	if !s.ECS.HasEntity(e) {
		t.Fatal("a projectile died on grass")
	}
}

func TestDamageableTileMustSayWhatItBecomes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atlas.json")
	if err := os.WriteFile(path, []byte(`[{"id": 1, "imagePath": "missing.png", "maxHealth": 10, "passable": true}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(testutil.RepoRoot())

	if err := tilesystem.NewChunkContainer().GenerateTileAtlasFromJson(path); err == nil {
		t.Fatal("a damageable tile without destroyedInto was accepted")
	}
}
