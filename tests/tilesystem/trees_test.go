package tilesystem_test

import (
	"ebittest/content/items"
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

// The closest tree to the start with nothing but grass and active chunks around it, so that what's near it can only
// touch the tree and grass
func findLoneTree(t *testing.T, s *testutil.Session) utils.Vec2i {
	t.Helper()
	lone := func(tile utils.Vec2i) bool {
		if f, ok := s.Chunks.FeatureAt(tile); !ok || f != data.Feature_Tree {
			return false
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				n := utils.Vec2i{X: tile.X + dx, Y: tile.Y + dy}
				if isRock(n) || !s.Chunks.IsActiveWorldPos(utils.TilePosToWorldPos(n)) {
					return false
				}
			}
		}
		return true
	}

	center := utils.WorldPosToTilePos(testutil.PlayerStart)
	for r := 0; r <= 20; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if tile := (utils.Vec2i{X: center.X + dx, Y: center.Y + dy}); max(utils.AbsInt(dx), utils.AbsInt(dy)) == r && lone(tile) {
					return tile
				}
			}
		}
	}
	t.Fatal("no lone tree in the active chunks")
	return utils.Vec2i{}
}

func addCircle(t *testing.T, x *ecs.ECSContainer, pos utils.Vec2f, radius float64, comps ...func(shapes.Shape) ecs.Component) common.EntityId {
	t.Helper()
	shape, err := shapes.NewCircleShape(radius, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	built := []ecs.Component{}
	for _, c := range comps {
		built = append(built, c(shape))
	}
	return testutil.AddMovable(t, x, pos, false, built...)
}

func resolveTerrain(t *testing.T, s *testutil.Session) {
	t.Helper()
	physics := ecs.Colliders(&s.ECS.PhysicsColliders)
	potential, err := s.Chunks.GetTilesWithPotentialPhysicsCollisions(s.ECS)
	if err != nil {
		t.Fatal(err)
	}
	aabb, err := tilesystem.GetAABBCollisions(physics, potential, s.ECS)
	if err != nil {
		t.Fatal(err)
	}
	cols, err := tilesystem.GetCollisions(physics, aabb, s.ECS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tilesystem.ResolveTileCollisions(cols, s.ECS); err != nil {
		t.Fatal(err)
	}
}

func TestTrunksBlockMovementButNotTheRestOfTheirTile(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := utils.TilePosToWorldPos(findLoneTree(t, s))
	mob := func(shape shapes.Shape) ecs.Component {
		return ecs.NewPhysicsColliderComponent(ecs.Layer_Player, ecs.Layer_Terrain, ecs.Collider_Mob, shape)
	}

	// The trunk is 8×6, centered 3px below the tile's center
	againstTrunk := tree.Add(utils.Vec2f{X: -6, Y: 3})
	pushed := addCircle(t, s.ECS, againstTrunk, 3, mob)
	aboveTrunk := tree.Add(utils.Vec2f{X: -4, Y: -6})
	free := addCircle(t, s.ECS, aboveTrunk, 3, mob)

	resolveTerrain(t, s)

	if got := testutil.WorldPos(t, s.ECS, pushed); got.X >= againstTrunk.X {
		t.Errorf("a mob overlapping the trunk from the left moved from %v to %v, want it pushed left", againstTrunk, got)
	}
	if got := testutil.WorldPos(t, s.ECS, free); got != aboveTrunk {
		t.Errorf("a mob on the tree's tile but clear of its trunk moved from %v to %v", aboveTrunk, got)
	}
}

func TestTrunksStopProjectiles(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := utils.TilePosToWorldPos(findLoneTree(t, s))
	projectile := func(shape shapes.Shape) ecs.Component {
		return ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, ecs.Layer_Terrain, shape)
	}
	dies := func(shapes.Shape) ecs.Component { return ecs.NewContactDamageComponent(-1, 0, 10).WithDieOnContact() }

	hits := addCircle(t, s.ECS, tree.Add(utils.Vec2f{Y: 3}), 1, projectile, dies)
	misses := addCircle(t, s.ECS, tree.Add(utils.Vec2f{X: -5, Y: -5}), 1, projectile, dies)

	touchTerrain(t, s)

	if s.ECS.HasEntity(hits) {
		t.Error("a projectile on a trunk survived")
	}
	if !s.ECS.HasEntity(misses) {
		t.Error("a projectile on the tree's tile but clear of its trunk died")
	}
}

func TestAnExplosionTurnsTheGrassUnderATreeIntoDirtAndTheTreeStays(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)

	addExplosion(t, s.ECS, utils.TilePosToWorldPos(tree), ecs.Layer_Terrain|ecs.Layer_Ground, []float64{4}, 50)
	touchTerrain(t, s)

	if got := currentTile(s, tree); got != data.Tile_Dirt {
		t.Fatalf("the grass under the tree is %v after a 50-damage explosion, want dirt", got)
	}
	if f, ok := s.Chunks.FeatureAt(tree); !ok || f != data.Feature_Tree {
		t.Fatal("the tree went with the grass under it")
	}
}

func TestHittingATrunkDamagesTheTreeButNotTheGroundUnderIt(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)
	groundBefore, _ := s.Chunks.TileHealth(tree)

	// Like a rocket's body: it collides with terrain, which the trunk is, but not with the ground
	shell := addExplosion(t, s.ECS, utils.TilePosToWorldPos(tree).Add(utils.Vec2f{Y: 3}), ecs.Layer_Terrain, []float64{1}, 30)
	cols := touchTerrain(t, s)

	hitTrunk := false
	for _, c := range cols {
		hitTrunk = hitTrunk || (c.E == shell && c.Tile == tree && c.Feature)
	}
	if !hitTrunk {
		t.Fatalf("collisions %+v don't say the trunk was hit", cols)
	}
	if health, _ := s.Chunks.FeatureHealth(tree); health != 70 {
		t.Fatalf("the tree has %d health after a 30-damage hit, want 70", health)
	}
	if after, _ := s.Chunks.TileHealth(tree); after != groundBefore {
		t.Fatalf("the grass under the trunk went from %d to %d health", groundBefore, after)
	}
}

func TestAnExplosionDamagesBothATreeAndTheGroundUnderItOnce(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)

	explosion := addExplosion(t, s.ECS, utils.TilePosToWorldPos(tree), ecs.Layer_Terrain|ecs.Layer_Ground, []float64{6}, 20)
	cd, _ := s.ECS.ContactDamages.Get(explosion)
	cd.WithActiveMs(0).WithRehitIntervalMs(0)
	for range 3 {
		touchTerrain(t, s)
	}

	if health, _ := s.Chunks.FeatureHealth(tree); health != 80 {
		t.Fatalf("the tree has %d health after an explosion of 20 touched it for 3 ticks, want 80: hit once", health)
	}
	if health, _ := s.Chunks.TileHealth(tree); health != 30 {
		t.Fatalf("the grass under it has %d health, want 30: hit once as well", health)
	}
}

func TestADestroyedTreeLeavesAStumpThatsOutOfTheWay(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)

	if !s.ECS.DamageFeature(tree, 60) || !s.ECS.DamageFeature(tree, 40) {
		t.Fatal("the tree couldn't be damaged")
	}
	if f, _ := s.Chunks.FeatureAt(tree); f != data.Feature_Stump {
		t.Fatalf("after 100 damage the tree is %v, want a stump", f)
	}
	if s.Chunks.CanDamageFeature(tree) || s.ECS.DamageFeature(tree, 10) {
		t.Fatal("the stump can be damaged")
	}

	center := utils.TilePosToWorldPos(tree)
	sprites := s.Chunks.FeatureSprites([2]utils.Vec2f{center, center})
	if len(sprites) != 1 || sprites[0].ImageAsset != common.AssetImageTreeStump {
		t.Fatalf("sprites on the tile are %+v, want the stump", sprites)
	}

	mob := addCircle(t, s.ECS, center.Add(utils.Vec2f{Y: 3}), 3, func(shape shapes.Shape) ecs.Component {
		return ecs.NewPhysicsColliderComponent(ecs.Layer_Player, ecs.Layer_Terrain, ecs.Collider_Mob, shape)
	})
	resolveTerrain(t, s)
	if got := testutil.WorldPos(t, s.ECS, mob); got != center.Add(utils.Vec2f{Y: 3}) {
		t.Fatalf("a mob where the trunk was moved to %v, so the stump still blocks", got)
	}
}

func TestTreesAreDrawnWhereTheyStand(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)
	center := utils.TilePosToWorldPos(tree)

	sprites := s.Chunks.FeatureSprites([2]utils.Vec2f{center.Subtract(utils.Vec2f{X: 8, Y: 8}), center.Add(utils.Vec2f{X: 8, Y: 8})})
	want := common.StaticSprite{ImageAsset: common.AssetImageTree, Pos: center.Add(utils.Vec2f{X: 1, Y: -7}), Layer: 20}
	if len(sprites) != 1 || sprites[0] != want {
		t.Fatalf("sprites on the tree's tile are %+v, want just %+v", sprites, want)
	}

	away := center.Add(utils.Vec2f{X: 1000})
	if sprites := s.Chunks.FeatureSprites([2]utils.Vec2f{away, away}); len(sprites) != 0 {
		t.Fatalf("sprites far from any active chunk: %+v", sprites)
	}
}

func TestDamagedTreesAndStumpsAreSaved(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	damaged := findLoneTree(t, s)
	felled := findTile(t, s, func(tile utils.Vec2i) bool {
		f, ok := s.Chunks.FeatureAt(tile)
		return ok && f == data.Feature_Tree && tile != damaged
	})

	if !s.ECS.DamageFeature(damaged, 30) || !s.ECS.DamageFeature(felled, 100) {
		t.Fatal("the trees couldn't be damaged")
	}

	s.Commit()
	r := testutil.StartSession(t)
	if health, ok := r.Chunks.FeatureHealth(damaged); !ok || health != 70 {
		t.Fatalf("after restarting, the damaged tree has %d health (%v), want 70", health, ok)
	}
	if f, _ := r.Chunks.FeatureAt(felled); f != data.Feature_Stump {
		t.Fatalf("after restarting, the felled tree is %v, want a stump", f)
	}
}

func TestTerrainBlocksTossesWhereItsImpassable(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := utils.TilePosToWorldPos(findLoneTree(t, s))
	rock := utils.TilePosToWorldPos(findTile(t, s, isRock))

	for _, c := range []struct {
		name string
		pos  utils.Vec2f
		want bool
	}{
		{"rock", rock, true},
		{"a trunk", tree.Add(utils.Vec2f{Y: 3}), true},
		{"a tree's tile beside its trunk", tree.Add(utils.Vec2f{X: -6, Y: -6}), false},
		{"grass", tree.Add(utils.Vec2f{X: data.TileSize}), false},
		{"outside the active chunks", tree.Add(utils.Vec2f{X: 100000}), true},
	} {
		if got := s.Chunks.IsBlockedAt(c.pos); got != c.want {
			t.Errorf("%s at %v: blocked %v, want %v", c.name, c.pos, got, c.want)
		}
	}
}

func TestFelledTreesDropPlanks(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)

	trees := []utils.Vec2i{}
	center := utils.WorldPosToTilePos(testutil.PlayerStart)
	for dy := -24; dy <= 24; dy++ {
		for dx := -24; dx <= 24; dx++ {
			tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
			if f, ok := s.Chunks.FeatureAt(tile); ok && f == data.Feature_Tree {
				trees = append(trees, tile)
			}
		}
	}
	if len(trees) < 5 {
		t.Fatalf("only %d trees in the active chunks, too few to see what they drop", len(trees))
	}

	dropCounts := map[int]int{}
	for _, tree := range trees {
		before := len(s.ECS.Items.GetEntities())
		if !s.ECS.DamageFeature(tree, 100) {
			t.Fatalf("couldn't fell the tree at %v", tree)
		}

		dropped := s.ECS.Items.GetEntities()[before:]
		dropCounts[len(dropped)]++
		for _, e := range dropped {
			item, _ := s.ECS.Items.Get(e)
			if item.Kind() != items.ItemKind_Planks || item.Count() != 1 || !s.ECS.Tosses.HasComponent(e) {
				t.Fatalf("the tree at %v dropped a %v of kind %v, tossed %v; want single planks, each tossed",
					tree, item.Count(), item.Kind(), s.ECS.Tosses.HasComponent(e))
			}
			if pos := testutil.WorldPos(t, s.ECS, e); pos != utils.TilePosToWorldPos(tree) {
				t.Fatalf("a plank was dropped at %v, want from the tree at %v", pos, utils.TilePosToWorldPos(tree))
			}
		}
	}

	for n := range dropCounts {
		if n > 2 {
			t.Fatalf("drop counts %v, want only 0, 1 or 2 planks per tree", dropCounts)
		}
	}
	if dropCounts[1] == 0 || dropCounts[2] == 0 {
		t.Fatalf("over %d trees, drop counts were %v; want some to drop one and some two", len(trees), dropCounts)
	}
}

func TestFeatureDropTablesMustAddUp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "features.json")
	atlas := `[{"id": 1, "imageAsset": "tree", "maxHealth": 10, "destroyedInto": 1,
		"drops": [{"chance": 60, "items": [{"kind": 1, "count": 1}]}, {"chance": 50, "items": [{"kind": 1, "count": 2}]}]}]`
	if err := os.WriteFile(path, []byte(atlas), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := tilesystem.NewChunkContainer().GenerateFeatureAtlasFromJson(path); err == nil {
		t.Fatal("a drop table adding up to 110% was accepted")
	}
}
