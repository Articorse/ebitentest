package tilesystem_test

import (
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"ebittest/worldgen"
	"testing"
)

func TestNewWorldSpawnIsTheClosestPassableTileCenter(t *testing.T) {
	testutil.TempWorld(t)
	cc := tilesystem.NewChunkContainer()
	if err := cc.GenerateTileAtlasFromJson(testutil.AtlasPath); err != nil {
		t.Fatal(err)
	}
	cc.SetSeed(testutil.WorldSeed)

	onRock, onGround := 0, 0
	for i := range 400 {
		start := utils.Vec2i{X: i%20*7 - 70, Y: i/20*7 - 70}
		if worldgen.Tile(testutil.WorldSeed, start) == data.Tile_Rock {
			onRock++
		} else {
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
		if worldgen.Tile(testutil.WorldSeed, got) == data.Tile_Rock {
			t.Fatalf("spawn for tile %v is on rock at %v", start, got)
		}

		gotDistSq := (got.X-start.X)*(got.X-start.X) + (got.Y-start.Y)*(got.Y-start.Y)
		for dy := -12; dy <= 12; dy++ {
			for dx := -12; dx <= 12; dx++ {
				tile := utils.Vec2i{X: start.X + dx, Y: start.Y + dy}
				if dx*dx+dy*dy < gotDistSq && worldgen.Tile(testutil.WorldSeed, tile) != data.Tile_Rock {
					t.Fatalf("spawn for tile %v is %v, but %v is closer and passable", start, got, tile)
				}
			}
		}
	}

	if onRock == 0 || onGround == 0 {
		t.Fatalf("%d starts on rock and %d on ground, want both", onRock, onGround)
	}
}
