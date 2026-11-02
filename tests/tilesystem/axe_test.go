package tilesystem_test

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
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
	if err := s.ECS.SelectInventorySlot(player, 3); err != nil {
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
