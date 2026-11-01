package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/damagesystem"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"ebittest/worldgen"
	"testing"
)

func findTile(t *testing.T, s *testutil.Session, want func(utils.Vec2i) bool) utils.Vec2i {
	t.Helper()
	center := utils.WorldPosToTilePos(testutil.PlayerStart)
	for r := 0; r <= 6; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
				if want(tile) && s.Chunks.IsActiveWorldPos(utils.TilePosToWorldPos(tile)) {
					return tile
				}
			}
		}
	}
	t.Fatal("no such tile near the start")
	return utils.Vec2i{}
}

func isRock(tile utils.Vec2i) bool {
	return worldgen.Tile(testutil.WorldSeed, tile) == data.Tile_Rock
}

func addProjectile(t *testing.T, x *ecs.ECSContainer, pos utils.Vec2f, mask ecs.LayerMask) common.EntityId {
	t.Helper()
	shape, err := shapes.NewCircleShape(5, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	return testutil.AddMovable(t, x, pos, false,
		ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, mask, shape),
		ecs.NewContactDamageComponent(-1, 0, true, false, 10))
}

func touchTerrain(t *testing.T, s *testutil.Session) []common.TileCollision {
	t.Helper()
	cols, err := s.Chunks.GetHurtboxTileCollisions(s.ECS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := damagesystem.DealContactDamage(nil, cols, s.Chunks, s.ECS); err != nil {
		t.Fatal(err)
	}
	testutil.RemoveScheduled(t, s.ECS)
	return cols
}

func TestProjectileDiesOnImpassableTiles(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	rock := findTile(t, s, isRock)
	open := findTile(t, s, func(tile utils.Vec2i) bool {
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if isRock(utils.Vec2i{X: tile.X + dx, Y: tile.Y + dy}) {
					return false
				}
			}
		}
		return true
	})

	hits := addProjectile(t, s.ECS, utils.TilePosToWorldPos(rock), ecs.Layer_Enemy|ecs.Layer_Terrain)
	misses := addProjectile(t, s.ECS, utils.TilePosToWorldPos(open), ecs.Layer_Enemy|ecs.Layer_Terrain)
	ignoresTerrain := addProjectile(t, s.ECS, utils.TilePosToWorldPos(rock), ecs.Layer_Enemy)

	cols := touchTerrain(t, s)

	if s.ECS.HasEntity(hits) {
		t.Error("a projectile on rock survived")
	}
	if !s.ECS.HasEntity(misses) {
		t.Error("a projectile away from rock died")
	}
	if !s.ECS.HasEntity(ignoresTerrain) {
		t.Error("a projectile whose mask leaves out terrain died on rock")
	}
	hitRock := false
	for _, c := range cols {
		hitRock = hitRock || (c.E == hits && c.Tile == rock)
	}
	if !hitRock {
		t.Errorf("collisions %+v don't say which rock tile was hit", cols)
	}
}

func TestProjectileAtTheEdgeOfTheActiveChunksDoesNotHitIt(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	active := func(tile utils.Vec2i) bool { return s.Chunks.IsActiveWorldPos(utils.TilePosToWorldPos(tile)) }

	start := utils.WorldPosToTilePos(testutil.PlayerStart)
	edge, found := utils.Vec2i{}, false
	for dy := -6; dy <= 6 && !found; dy++ {
		edge = utils.Vec2i{X: start.X, Y: start.Y + dy}
		for active(utils.Vec2i{X: edge.X + 1, Y: edge.Y}) {
			edge.X++
		}

		found = active(edge)
		for ny := -1; ny <= 1; ny++ {
			for nx := -1; nx <= 0; nx++ {
				found = found && !isRock(utils.Vec2i{X: edge.X + nx, Y: edge.Y + ny})
			}
		}
	}
	if !found {
		t.Fatal("no edge of the active chunks without rock next to it")
	}

	// Still in the edge tile, but reaching into the unloaded one next to it
	e := addProjectile(t, s.ECS, utils.TilePosToWorldPos(edge).Add(utils.Vec2f{X: 6}), ecs.Layer_Enemy|ecs.Layer_Terrain)
	if cols := touchTerrain(t, s); len(cols) != 0 {
		t.Fatalf("got terrain collisions %+v at the edge of the active chunks", cols)
	}

	if !s.ECS.HasEntity(e) {
		t.Fatal("a projectile at the edge of the active chunks died as if it had hit a wall")
	}
}

func TestMobsAreStillPushedOutOfRock(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	rock := findTile(t, s, func(tile utils.Vec2i) bool {
		return isRock(tile) && !isRock(utils.Vec2i{X: tile.X - 1, Y: tile.Y})
	})

	shape, err := shapes.NewCircleShape(5, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	start := utils.TilePosToWorldPos(rock).Subtract(utils.Vec2f{X: 10})
	mob := testutil.AddMovable(t, s.ECS, start, false,
		ecs.NewPhysicsColliderComponent(ecs.Layer_Player, ecs.Layer_Terrain, ecs.Collider_Mob, shape))

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

	if got := testutil.WorldPos(t, s.ECS, mob); got.X >= start.X {
		t.Fatalf("mob overlapping rock from the left moved from %v to %v, want it pushed left", start, got)
	}
}
