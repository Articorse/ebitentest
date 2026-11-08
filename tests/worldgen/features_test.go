package worldgen_test

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen"
	"maps"
	"testing"
)

func TestTreesGrowOnGrassAndNeverNextToEachOther(t *testing.T) {
	const seed = 99
	trees := 0
	for y := -100; y < 100; y++ {
		for x := -100; x < 100; x++ {
			tile := utils.Vec2i{X: x, Y: y}
			if worldgen.Feature(seed, tile) != data.Feature_Tree {
				continue
			}
			trees++

			if worldgen.Tile(seed, tile).Top() != data.Tile_Grass {
				t.Fatalf("tree at %v grows on %v", tile, worldgen.Tile(seed, tile).Top())
			}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					n := utils.Vec2i{X: x + dx, Y: y + dy}
					if n != tile && worldgen.Feature(seed, n) == data.Feature_Tree {
						t.Fatalf("trees at %v and %v are next to each other", tile, n)
					}
				}
			}
		}
	}

	if trees == 0 {
		t.Fatal("no trees")
	}
}

func TestTreesFormForestsAndClearings(t *testing.T) {
	const seed, window = 99, 16
	densest, emptyWindows := 0.0, 0
	for wy := -10; wy < 10; wy++ {
		for wx := -10; wx < 10; wx++ {
			grass, trees := 0, 0
			for y := range window {
				for x := range window {
					tile := utils.Vec2i{X: wx*window + x, Y: wy*window + y}
					if worldgen.Tile(seed, tile).Top() != data.Tile_Grass {
						continue
					}
					grass++
					if worldgen.Feature(seed, tile) == data.Feature_Tree {
						trees++
					}
				}
			}

			if grass < window*window/2 {
				continue
			}
			densest = max(densest, float64(trees)/float64(grass))
			if trees == 0 {
				emptyWindows++
			}
		}
	}

	if densest < 0.15 || emptyWindows == 0 {
		t.Fatalf("densest grassy %d×%d area is %.0f%% trees and %d have none; want forests of at least 15%% and clearings",
			window, window, densest*100, emptyWindows)
	}
}

func TestChunkFeaturesMatchFeaturesOneByOne(t *testing.T) {
	const seed = 99
	for cy := -6; cy < 6; cy++ {
		for cx := -6; cx < 6; cx++ {
			chunkPos := utils.Vec2i{X: cx, Y: cy}
			want := map[utils.Vec2i]data.FeatureEnum{}
			for y := range data.ChunkSize {
				for x := range data.ChunkSize {
					tile := utils.Vec2i{X: cx*data.ChunkSize + x, Y: cy*data.ChunkSize + y}
					if f := worldgen.Feature(seed, tile); f != data.Feature_None {
						want[tile] = f
					}
				}
			}

			if got := worldgen.ChunkFeatures(seed, chunkPos); !maps.Equal(got, want) {
				t.Fatalf("chunk %v has features %v, but one by one they're %v", chunkPos, got, want)
			}
		}
	}
}
