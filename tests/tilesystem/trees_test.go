package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
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

func TestHittingATrunkDoesntDamageTheGroundUnderIt(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)
	before, _ := s.Chunks.TileHealth(tree)

	// Like a rocket's body: it collides with terrain, which the trunk is, but not with the ground
	shell := addExplosion(t, s.ECS, utils.TilePosToWorldPos(tree).Add(utils.Vec2f{Y: 3}), ecs.Layer_Terrain, []float64{1}, 50)
	cols := touchTerrain(t, s)

	hitTrunk := false
	for _, c := range cols {
		hitTrunk = hitTrunk || (c.E == shell && c.Tile == tree && c.Feature)
	}
	if !hitTrunk {
		t.Fatalf("collisions %+v don't say the trunk was hit", cols)
	}
	if after, _ := s.Chunks.TileHealth(tree); after != before {
		t.Fatalf("the grass under the trunk went from %d to %d health", before, after)
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

func TestTreesStayAfterTheirChunkIsSavedAndLoaded(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)

	// Damaging the grass next to the tree makes its chunk get saved, rather than generated again
	chunkOf := func(tile utils.Vec2i) utils.Vec2i { return utils.WorldPosToChunkGridPos(utils.TilePosToWorldPos(tile)) }
	neighbor := tree
	for _, n := range []utils.Vec2i{{X: tree.X - 1, Y: tree.Y}, {X: tree.X + 1, Y: tree.Y}, {X: tree.X, Y: tree.Y - 1}, {X: tree.X, Y: tree.Y + 1}} {
		if chunkOf(n) == chunkOf(tree) {
			neighbor = n
			break
		}
	}
	if neighbor == tree || !s.Chunks.DamageTile(neighbor, 10) {
		t.Fatal("couldn't damage the grass next to the tree")
	}

	s.Commit()
	r := testutil.StartSession(t)
	if f, ok := r.Chunks.FeatureAt(tree); !ok || f != data.Feature_Tree {
		t.Fatalf("after restarting, tile %v has feature %v (%v), want a tree", tree, f, ok)
	}
	if health, ok := r.Chunks.TileHealth(neighbor); !ok || health != 40 {
		t.Fatalf("after restarting, the damaged grass has %d health (%v), so its chunk wasn't the saved one", health, ok)
	}
}

func TestTheTerrainKnowsWhichTilesHaveTrees(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)

	if !s.Chunks.HasFeature(tree) {
		t.Fatal("the tree's tile has no feature")
	}
	if beside := (utils.Vec2i{X: tree.X + 1, Y: tree.Y}); s.Chunks.HasFeature(beside) {
		t.Fatalf("the tile beside the tree, %v, has a feature", beside)
	}
}
