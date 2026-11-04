package worldgen_test

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen"
	"slices"
	"sync"
	"testing"
)

const waterSeed, waterSide = 7, 1024

// A square of generated tiles centered on the origin, shared between the tests since it takes a while
var waterRegion = sync.OnceValue(func() []data.TileEnum {
	tiles := make([]data.TileEnum, waterSide*waterSide)
	for y := range waterSide {
		for x := range waterSide {
			tiles[y*waterSide+x] = worldgen.Tile(waterSeed, regionTile(y*waterSide+x))
		}
	}
	return tiles
})

func regionTile(i int) utils.Vec2i {
	return utils.Vec2i{X: i%waterSide - waterSide/2, Y: i/waterSide - waterSide/2}
}

// The sizes of the 4-connected patches of tiles that are in, and whether each touches a tile that's in touching
func patches(tiles []data.TileEnum, in, touching func(data.TileEnum) bool) (sizes []int, touches []bool) {
	seen := make([]bool, len(tiles))
	for i := range tiles {
		if seen[i] || !in(tiles[i]) {
			continue
		}
		size, touch := 0, false
		stack := []int{i}
		seen[i] = true
		for len(stack) > 0 {
			c := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			size++
			cx, cy := c%waterSide, c/waterSide
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				x, y := cx+d[0], cy+d[1]
				if x < 0 || y < 0 || x >= waterSide || y >= waterSide {
					continue
				}
				j := y*waterSide + x
				touch = touch || touching(tiles[j])
				if !seen[j] && in(tiles[j]) {
					seen[j] = true
					stack = append(stack, j)
				}
			}
		}
		sizes = append(sizes, size)
		touches = append(touches, touch)
	}
	return sizes, touches
}

func isDeep(t data.TileEnum) bool    { return t == data.Tile_WaterDeep }
func isShallow(t data.TileEnum) bool { return t == data.Tile_WaterShallow }

func TestDeepWaterIsAlwaysSurroundedByShallowWater(t *testing.T) {
	tiles := waterRegion()
	for i, tile := range tiles {
		x, y := i%waterSide, i/waterSide
		if !isDeep(tile) || x == 0 || y == 0 || x == waterSide-1 || y == waterSide-1 {
			continue
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if n := tiles[(y+dy)*waterSide+x+dx]; !worldgen.IsWater(n) {
					t.Fatalf("deep water at %v is next to %v", regionTile(i), n)
				}
			}
		}
	}
}

func TestLakesRangeFromADozenTilesToThousandsAndPondsAreShallow(t *testing.T) {
	tiles := waterRegion()
	lakes, _ := patches(tiles, isDeep, isDeep)
	slices.Sort(lakes)
	if len(lakes) == 0 || lakes[0] > 20 || lakes[len(lakes)-1] < 2000 {
		t.Fatalf("lakes of deep water range over %v tiles, want from a dozen or so to thousands", lakes)
	}

	// Lakes too small to keep any deep water leave a few behind as well
	_, withDeep := patches(tiles, isShallow, isDeep)
	ponds := 0
	for _, deep := range withDeep {
		if !deep {
			ponds++
		}
	}
	if ponds < 200 {
		t.Fatalf("%d patches of shallow water are on their own, want ponds scattered about", ponds)
	}
}

func TestWaterCoversAboutATenthOfTheLand(t *testing.T) {
	water, land := 0, 0
	for _, tile := range waterRegion() {
		if tile == data.Tile_Rock || tile == data.Tile_RockFloor {
			continue
		}
		land++
		if worldgen.IsWater(tile) {
			water++
		}
	}
	if share := float64(water) / float64(land); share < 0.06 || share > 0.14 {
		t.Fatalf("water covers %.1f%% of the land, want about 10%%", share*100)
	}
}

func TestMountainsStayDryAndWaterGrowsNoTrees(t *testing.T) {
	for i, tile := range waterRegion() {
		if !worldgen.IsWater(tile) {
			continue
		}
		at := regionTile(i)
		if worldgen.Roof(waterSeed, at) != data.Roof_None {
			t.Fatalf("water at %v is under a mountain's roof", at)
		}
		if f := worldgen.Feature(waterSeed, at); f != data.Feature_None {
			t.Fatalf("water at %v has feature %v", at, f)
		}
	}
}
