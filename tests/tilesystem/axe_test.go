package tilesystem_test

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

// Slashes toward the target with the starting axe of a player standing 18px left of it
func slashFromTheLeft(t *testing.T, s *testutil.Session, target utils.Vec2f) common.EntityId {
	t.Helper()
	player, err := templates.SpawnPlayer(s.ECS)
	if err != nil {
		t.Fatal(err)
	}
	testutil.SetPos(t, s.ECS, player, target.Subtract(utils.Vec2f{X: 18}))
	if err := templates.SpawnStartingEquipment(s.ECS, player); err != nil {
		t.Fatal(err)
	}
	if err := s.ECS.SelectHotbarSlot(player, 3); err != nil {
		t.Fatal(err)
	}

	used, err := s.ECS.ActivateEquipmentAbility(player, ecs.Equip_MainHand, ecs.AbilityTarget{Pos: target, HasPos: true}, 0)
	if err != nil || !used {
		t.Fatalf("slashing: %v, %v", used, err)
	}
	return player
}

func TestTheAxeCutsTreesButNotTheGroundUnderThem(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)
	groundBefore, _ := s.Chunks.TileHealth(tree)

	slashFromTheLeft(t, s, utils.TilePosToWorldPos(tree).Add(utils.Vec2f{Y: 3}))
	touchTerrain(t, s)
	touchTerrain(t, s)

	if health, _ := s.Chunks.FeatureHealth(tree); health != 50 {
		t.Fatalf("the tree has %d health after a slash, want 50: hit once for 50", health)
	}
	if after, _ := s.Chunks.TileHealth(tree); after != groundBefore {
		t.Fatalf("the ground under the tree went from %d to %d health", groundBefore, after)
	}
}

func TestTheAxeLeavesWallsAlone(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	rock := tileOf(t, s, data.Tile_Rock)

	slashFromTheLeft(t, s, utils.TilePosToWorldPos(rock))
	if cols := touchTerrain(t, s); len(cols) != 0 {
		t.Fatalf("the slash collided with %+v", cols)
	}
	if health, _ := s.Chunks.TileHealth(rock); health != 200 {
		t.Fatalf("the rock has %d health after a slash, want 200", health)
	}
}

func TestThePlayerIsBlockedByTrunks(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := utils.TilePosToWorldPos(findLoneTree(t, s))
	player, err := templates.SpawnPlayer(s.ECS)
	if err != nil {
		t.Fatal(err)
	}

	// The trunk is 8×6, centered 3px below the tile's center
	againstTrunk := tree.Add(utils.Vec2f{X: -6, Y: 3})
	testutil.SetPos(t, s.ECS, player, againstTrunk)
	resolveTerrain(t, s)

	if got := testutil.WorldPos(t, s.ECS, player); got.X >= againstTrunk.X {
		t.Fatalf("the player overlapping a trunk from the left moved from %v to %v, want it pushed left", againstTrunk, got)
	}
}

func TestTurnedHurtboxesHitTheTilesTheyCover(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	rock := tileOf(t, s, data.Tile_Rock)
	bar, err := shapes.NewPolygonShape([]utils.Vec2f{{X: 0, Y: -1}, {X: 3 * data.TileSize, Y: -1}, {X: 3 * data.TileSize, Y: 1}, {X: 0, Y: 1}}, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}

	// Pointing up from two tiles below the rock, so that only turned does it reach it
	e := testutil.AddMovable(t, s.ECS, utils.TilePosToWorldPos(utils.Vec2i{X: rock.X, Y: rock.Y + 2}), false,
		ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, ecs.Layer_Wall, bar),
		ecs.NewContactDamageComponent(-1, 0, 10))
	tr, _ := s.ECS.Transforms.Get(e)
	tr.SetLocalRotation(-math.Pi / 2)

	hit := false
	for _, c := range touchTerrain(t, s) {
		hit = hit || (c.E == e && c.Tile == rock)
	}
	if !hit {
		t.Fatal("a bar turned to point up at the rock didn't hit it")
	}
}
