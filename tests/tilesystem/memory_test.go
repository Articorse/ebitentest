package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"ebittest/worldgen"
	"testing"
)

func see(s *testutil.Session, viewer utils.Vec2f) *ecs.Vision {
	v := ecs.NewVisionComponent()
	origin := utils.WorldPosToTilePos(viewer)
	s.ECS.LightAround(origin, data.VisionMaxRangeTiles, v.Light())
	v.Update(origin, s.Chunks.IsOpaqueTile, v.Light().At)
	s.Chunks.RememberSeenTiles(v)
	return v
}

func seeLitBy(s *testutil.Session, viewer utils.Vec2f, lightAt func(utils.Vec2i) uint8) *ecs.Vision {
	v := ecs.NewVisionComponent()
	v.Update(utils.WorldPosToTilePos(viewer), s.Chunks.IsOpaqueTile, lightAt)
	s.Chunks.RememberSeenTiles(v)
	return v
}

// The center of the closest tile to the start that's in daylight, since the start itself may be inside a mountain
func outside(t *testing.T, s *testutil.Session) utils.Vec2f {
	t.Helper()
	return utils.TilePosToWorldPos(findTile(t, s, func(tile utils.Vec2i) bool {
		return s.Chunks.IsOpenToSky(tile) && !s.Chunks.IsOpaqueTile(tile)
	}))
}

func TestExactlyTheSeenTilesAreRemembered(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	viewer := outside(t, s).Add(utils.Vec2f{X: 5, Y: -3})
	v := see(s, viewer)

	remembered, unseen := 0, 0
	center := utils.WorldPosToTilePos(viewer)
	for dy := -data.VisionMaxRangeTiles - 2; dy <= data.VisionMaxRangeTiles+2; dy++ {
		for dx := -data.VisionMaxRangeTiles - 2; dx <= data.VisionMaxRangeTiles+2; dx++ {
			tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
			got, ok := s.Chunks.RecallTile(tile)
			if ok != v.CanSee(tile) {
				t.Fatalf("tile %v remembered %v, but seen %v", tile, ok, v.CanSee(tile))
			}
			if ok && got != worldgen.Tile(testutil.WorldSeed, tile) {
				t.Fatalf("tile %v remembered as %v, but it's %v", tile, got, worldgen.Tile(testutil.WorldSeed, tile))
			}
			if ok {
				remembered++
			} else {
				unseen++
			}
		}
	}
	if remembered == 0 || unseen == 0 {
		t.Fatalf("%d tiles remembered and %d not, want some of each", remembered, unseen)
	}
}

func TestMemorySurvivesARestart(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	start := outside(t, s)
	see(s, start)
	tile := utils.WorldPosToTilePos(start)
	want, ok := s.Chunks.RecallTile(tile)
	if !ok {
		t.Fatal("the viewer's own tile isn't remembered")
	}
	s.Commit()

	r := testutil.StartSession(t)
	if got, ok := r.Chunks.RecallTile(tile); !ok || got != want {
		t.Fatalf("after restarting, tile %v remembered %v as %v, want %v", tile, ok, got, want)
	}
}

func TestRememberingDuringACommitIsSavedByTheNext(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	start := outside(t, s)
	see(s, start)
	before := rememberedAround(s, start)

	s.StartCommit()
	// Away from the mountain the start is beside, so that there's open land to see
	moved := start.Add(utils.Vec2f{X: -8 * data.TileSize, Y: 2 * data.TileSize})
	see(s, moved)
	for s.Chunks.IsCommitting() {
		s.Tick(1)
	}

	during := []utils.Vec2i{}
	for tile := range rememberedAround(s, start) {
		if !before[tile] {
			during = append(during, tile)
		}
	}
	if len(during) == 0 {
		t.Fatal("moving remembered nothing new")
	}

	s.Commit()
	r := testutil.StartSession(t)
	for _, tile := range during {
		if _, ok := r.Chunks.RecallTile(tile); !ok {
			t.Fatalf("tile %v, remembered while a commit was in flight, wasn't saved by the next", tile)
		}
	}
}

func rememberedAround(s *testutil.Session, pos utils.Vec2f) map[utils.Vec2i]bool {
	const reach = 2 * data.VisionMaxRangeTiles
	center := utils.WorldPosToTilePos(pos)
	remembered := map[utils.Vec2i]bool{}
	for dy := -reach; dy <= reach; dy++ {
		for dx := -reach; dx <= reach; dx++ {
			tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
			if _, ok := s.Chunks.RecallTile(tile); ok {
				remembered[tile] = true
			}
		}
	}
	return remembered
}

func TestTreesAreRememberedAsTheyWereLastSeen(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)

	if _, ok := s.Chunks.RecallFeature(tree); ok {
		t.Fatal("a tree is remembered before it was seen")
	}

	treePos := utils.TilePosToWorldPos(tree)
	see(s, treePos)
	if f, ok := s.Chunks.RecallFeature(tree); !ok || f != data.Feature_Tree {
		t.Fatalf("after seeing it, the tree is remembered as %v (%v)", f, ok)
	}
	if _, ok := s.Chunks.RecallFeature(utils.Vec2i{X: tree.X + 1, Y: tree.Y}); ok {
		t.Fatal("the grass beside the tree is remembered with a feature")
	}

	if !s.ECS.DamageFeature(tree, 100) {
		t.Fatal("couldn't fell the tree")
	}
	if f, _ := s.Chunks.RecallFeature(tree); f != data.Feature_Tree {
		t.Fatalf("the felled tree is remembered as %v before it was seen again, want the tree", f)
	}

	see(s, treePos)
	if f, _ := s.Chunks.RecallFeature(tree); f != data.Feature_Stump {
		t.Fatalf("after seeing it again, the felled tree is remembered as %v, want a stump", f)
	}
}

func TestRememberedTreesSurviveARestartAndAreDrawn(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)
	treePos := utils.TilePosToWorldPos(tree)
	see(s, treePos)
	s.Commit()

	r := testutil.StartSession(t)
	if f, ok := r.Chunks.RecallFeature(tree); !ok || f != data.Feature_Tree {
		t.Fatalf("after restarting, the tree is remembered as %v (%v)", f, ok)
	}

	sprites := r.Chunks.RememberedFeatureSprites([2]utils.Vec2f{treePos, treePos})
	want := common.StaticSprite{ImageAsset: common.AssetImageTree, Pos: treePos.Add(utils.Vec2f{X: 1, Y: -7}), Layer: 20}
	if len(sprites) != 1 || sprites[0] != want {
		t.Fatalf("remembered sprites on the tree's tile are %+v, want %+v", sprites, want)
	}

	away := treePos.Add(utils.Vec2f{X: 3000})
	if sprites := r.Chunks.RememberedFeatureSprites([2]utils.Vec2f{away, away}); len(sprites) != 0 {
		t.Fatalf("remembered sprites where nothing was seen: %+v", sprites)
	}
}
