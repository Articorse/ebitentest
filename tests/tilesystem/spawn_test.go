package tilesystem_test

import (
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"ebittest/worldgen"
	"testing"
)

func TestNewWorldSpawnIsTheClosestFreeTileCenter(t *testing.T) {
	testutil.TempWorld(t)
	cc := tilesystem.NewChunkContainer()
	if err := cc.GenerateTileAtlasFromJson(testutil.TileAtlasPath, testutil.ItemAtlas(t), testutil.Assets(t)); err != nil {
		t.Fatal(err)
	}
	cc.SetSeed(testutil.WorldSeed)

	free := func(t utils.Vec2i) bool {
		tiles := worldgen.Tile(testutil.WorldSeed, t)
		return tiles.Top() != data.Tile_Rock && !worldgen.IsWater(tiles) && worldgen.Feature(testutil.WorldSeed, t) == data.Feature_None
	}

	onRock, onTree, onGround := 0, 0, 0
	for i := range 400 {
		start := utils.Vec2i{X: i%20*7 - 70, Y: i/20*7 - 70}
		switch {
		case worldgen.Tile(testutil.WorldSeed, start).Top() == data.Tile_Rock:
			onRock++
		case !free(start):
			onTree++
		default:
			onGround++
		}

		pos, ok := cc.NewWorldSpawnPos(utils.Vec2f{X: float64(start.X * data.TileSize), Y: float64(start.Y * data.TileSize)})
		if !ok {
			t.Fatalf("no spawn found near tile %v", start)
		}

		got := utils.Vec2i{X: int(pos.X) / data.TileSize, Y: int(pos.Y) / data.TileSize}
		if pos != (utils.Vec2f{X: float64(got.X * data.TileSize), Y: float64(got.Y * data.TileSize)}) {
			t.Fatalf("spawn %v isn't a tile center", pos)
		}
		if !free(got) {
			t.Fatalf("spawn for tile %v is on rock, water or a tree at %v", start, got)
		}

		gotDistSq := (got.X-start.X)*(got.X-start.X) + (got.Y-start.Y)*(got.Y-start.Y)
		for dy := -12; dy <= 12; dy++ {
			for dx := -12; dx <= 12; dx++ {
				tile := utils.Vec2i{X: start.X + dx, Y: start.Y + dy}
				if dx*dx+dy*dy < gotDistSq && free(tile) {
					t.Fatalf("spawn for tile %v is %v, but %v is closer and free", start, got, tile)
				}
			}
		}
	}

	if onRock == 0 || onTree == 0 || onGround == 0 {
		t.Fatalf("%d starts on rock, %d on trees and %d on free ground, want some of each", onRock, onTree, onGround)
	}
}

func TestNewWorldsDontSpawnInWater(t *testing.T) {
	testutil.TempWorld(t)
	cc := tilesystem.NewChunkContainer()
	if err := cc.GenerateTileAtlasFromJson(testutil.TileAtlasPath, testutil.ItemAtlas(t), testutil.Assets(t)); err != nil {
		t.Fatal(err)
	}
	cc.SetSeed(testutil.WorldSeed)

	checked := 0
	for y := -400; y <= 400 && checked < 20; y += 3 {
		for x := -400; x <= 400 && checked < 20; x += 3 {
			start := utils.Vec2i{X: x, Y: y}
			if !worldgen.IsWater(worldgen.Tile(testutil.WorldSeed, start)) {
				continue
			}
			checked++
			pos, ok := cc.NewWorldSpawnPos(utils.TilePosToWorldPos(start))
			if tile := worldgen.Tile(testutil.WorldSeed, utils.WorldPosToTilePos(pos)); !ok || worldgen.IsWater(tile) {
				t.Fatalf("starting in water at %v, the spawn is at %v on %v (%v), want dry land", start, pos, tile, ok)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no water to start in near the origin")
	}
}
