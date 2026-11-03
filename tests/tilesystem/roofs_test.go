package tilesystem_test

import (
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"ebittest/worldgen"
	"testing"
)

func TestMountainsStayRoofedWhenDugIntoAndAfterARestart(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	rock := tileOf(t, s, data.Tile_Rock)
	grass := tileOf(t, s, data.Tile_Grass)

	if roof, ok := s.Chunks.RoofAt(rock); !ok || roof != data.Roof_Rock {
		t.Fatalf("rock is roofed with %v (%v), want a rock roof", roof, ok)
	}
	if roof, _ := s.Chunks.RoofAt(grass); roof != data.Roof_None {
		t.Fatalf("grass is roofed with %v, want none", roof)
	}

	if ok, _ := s.Chunks.DamageTile(rock, 1000); !ok {
		t.Fatal("the rock couldn't be mined")
	}
	if roof, _ := s.Chunks.RoofAt(rock); roof != data.Roof_Rock {
		t.Fatalf("mined rock is roofed with %v, want the mountain's roof to stay", roof)
	}

	s.Commit()
	r := testutil.StartSession(t)
	if roof, _ := r.Chunks.RoofAt(rock); roof != data.Roof_Rock {
		t.Fatalf("after restarting, the dug-out tile is roofed with %v, want a rock roof", roof)
	}
}

func TestRoofsAreRemembered(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	open := findTile(t, s, func(tile utils.Vec2i) bool {
		roof, _ := s.Chunks.RoofAt(tile)
		return roof == data.Roof_Rock && !s.Chunks.IsOpaqueTile(tile)
	})
	// Roofed tiles are dark until something lights them
	seeLitBy(s, utils.TilePosToWorldPos(open), func(utils.Vec2i) uint8 { return data.MaxLight })
	s.Commit()

	r := testutil.StartSession(t)
	if roof := r.Chunks.RecallRoof(open); roof != data.Roof_Rock {
		t.Fatalf("a seen tile under a roof is remembered with %v, want a rock roof", roof)
	}
	if roof := r.Chunks.RecallRoof(utils.Vec2i{X: open.X + 10000}); roof != data.Roof_None {
		t.Fatalf("a tile never seen is remembered with roof %v", roof)
	}
}

// A chunk that's never been saved is written for the first time when an entity moves into it, and must be written whole
func TestAChunkFirstSavedForAnEntityKeepsItsTreesAndRoofs(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	e := testutil.AddMovable(t, s.ECS, nearPlayer, true)
	testutil.SetPos(t, s.ECS, e, farAway)
	s.Tick(3)
	s.Commit()

	chunk := utils.WorldPosToChunkGridPos(farAway)
	tree, roofed := utils.Vec2i{}, utils.Vec2i{}
	foundTree, foundRoof := false, false
	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			t := utils.Vec2i{X: chunk.X*data.ChunkSize + x, Y: chunk.Y*data.ChunkSize + y}
			if !foundTree && worldgen.Feature(testutil.WorldSeed, t) == data.Feature_Tree {
				tree, foundTree = t, true
			}
			if !foundRoof && worldgen.Roof(testutil.WorldSeed, t) != data.Roof_None {
				roofed, foundRoof = t, true
			}
		}
	}
	if !foundTree || !foundRoof {
		t.Fatalf("the chunk at %v has a tree %v and a roof %v, want both to check", chunk, foundTree, foundRoof)
	}

	s.MovePlayer(farAway)
	s.Tick(5)
	if f, ok := s.Chunks.FeatureAt(tree); !ok || f != data.Feature_Tree {
		t.Fatalf("the tree at %v is %v (%v) after its chunk was first saved for an entity", tree, f, ok)
	}
	if roof, _ := s.Chunks.RoofAt(roofed); roof == data.Roof_None {
		t.Fatalf("the roof at %v is gone after its chunk was first saved for an entity", roofed)
	}
}
