package worldgen_test

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen"
	"ebittest/worldgen/noise"
	"math"
	"testing"
)

func TestNoiseIsBoundedSmoothAndSeeded(t *testing.T) {
	differs := false
	for i := range 200 {
		for j := range 200 {
			x, y := float64(i-100)*0.173, float64(j-100)*0.131
			v := noise.OpenSimplex2(42, x, y)
			if v < -1 || v > 1 {
				t.Fatalf("noise at (%v, %v) is %v, outside [-1, 1]", x, y, v)
			}
			if step := math.Abs(v - noise.OpenSimplex2(42, x+1e-3, y)); step > 0.02 {
				t.Fatalf("noise jumps by %v over a step of 0.001 at (%v, %v)", step, x, y)
			}
			if v != noise.OpenSimplex2(42, x, y) {
				t.Fatalf("noise at (%v, %v) isn't repeatable", x, y)
			}
			differs = differs || v != noise.OpenSimplex2(43, x, y)
		}
	}
	if !differs {
		t.Fatal("a different seed gives the same noise")
	}
}

func TestChunkTilesMatchTheirWorldPositions(t *testing.T) {
	const seed = 7
	for _, chunkPos := range []utils.Vec2i{{X: 0, Y: 0}, {X: -3, Y: 5}, {X: 12, Y: -9}} {
		tiles := worldgen.ChunkTiles(seed, chunkPos)
		for y := range data.ChunkSize {
			for x := range data.ChunkSize {
				pos := utils.Vec2i{X: chunkPos.X*data.ChunkSize + x, Y: chunkPos.Y*data.ChunkSize + y}
				if got, want := tiles[y*data.ChunkSize+x], worldgen.Tile(seed, pos); got != want {
					t.Fatalf("chunk %v has %v at tile %v, want %v", chunkPos, got, pos, want)
				}
			}
		}
	}
}

func TestTerrainFormsPatches(t *testing.T) {
	const size = 200
	kinds := map[data.TileEnum]bool{}
	same, pairs := 0, 0
	for y := range size {
		for x := range size {
			tile := worldgen.Tile(1, utils.Vec2i{X: x, Y: y})
			kinds[tile] = true
			if x > 0 {
				pairs++
				if tile == worldgen.Tile(1, utils.Vec2i{X: x - 1, Y: y}) {
					same++
				}
			}
		}
	}

	if len(kinds) < 2 {
		t.Fatalf("only %v appear", kinds)
	}
	// Uncorrelated tiles would agree with their neighbour far less often
	if agreement := float64(same) / float64(pairs); agreement < 0.85 {
		t.Fatalf("only %.2f of neighbouring tiles match, so the terrain isn't forming patches", agreement)
	}
}

func TestNearestCellMatchesBruteForce(t *testing.T) {
	const seed = 99
	for i := range 2000 {
		x := float64(i%50)*0.37 - 9
		y := float64(i/50)*0.53 - 11

		wantX, wantY, best := int64(0), int64(0), math.Inf(1)
		for cy := int64(math.Floor(y)) - 4; cy <= int64(math.Floor(y))+4; cy++ {
			for cx := int64(math.Floor(x)) - 4; cx <= int64(math.Floor(x))+4; cx++ {
				px, py := noise.FeaturePoint(seed, cx, cy)
				if d := (px-x)*(px-x) + (py-y)*(py-y); d < best {
					best, wantX, wantY = d, cx, cy
				}
			}
		}

		if gotX, gotY := noise.NearestCell(seed, x, y); gotX != wantX || gotY != wantY {
			t.Fatalf("nearest cell to (%v, %v) is (%d, %d), want (%d, %d)", x, y, gotX, gotY, wantX, wantY)
		}
	}
}

func TestHashDoesNotConfuseCoordinates(t *testing.T) {
	for _, pair := range [][2][2]int64{{{-1, 1}, {1, -1}}, {{1, 2}, {2, 1}}, {{-1, -1}, {1, 1}}, {{0, 5}, {5, 0}}} {
		a, b := pair[0], pair[1]
		if noise.Hash2(3, a[0], a[1]) == noise.Hash2(3, b[0], b[1]) {
			t.Errorf("(%d, %d) and (%d, %d) hash alike", a[0], a[1], b[0], b[1])
		}
	}
}

func TestSeedChangesTheTerrain(t *testing.T) {
	differing := 0
	for x := range 400 {
		pos := utils.Vec2i{X: x * 7, Y: x * 3}
		if worldgen.Tile(1, pos) != worldgen.Tile(2, pos) {
			differing++
		}
	}
	if differing < 100 {
		t.Fatalf("only %d of 400 tiles differ between two seeds", differing)
	}
}
