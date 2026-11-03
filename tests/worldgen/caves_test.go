package worldgen_test

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen"
	"testing"
)

const caveAreaSize = 600

func caveArea(seed uint64) []data.TileEnum {
	tiles := make([]data.TileEnum, caveAreaSize*caveAreaSize)
	for y := range caveAreaSize {
		for x := range caveAreaSize {
			tiles[y*caveAreaSize+x] = worldgen.Tile(seed, utils.Vec2i{X: x, Y: y})
		}
	}
	return tiles
}

// The tiles reachable through the passable ones from the given starts, moving in the four directions
func flood(tiles []data.TileEnum, starts []int, through func(data.TileEnum) bool) []bool {
	reached := make([]bool, len(tiles))
	queue := []int{}
	for _, i := range starts {
		reached[i] = true
		queue = append(queue, i)
	}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%caveAreaSize, i/caveAreaSize
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= caveAreaSize || ny >= caveAreaSize {
				continue
			}
			if j := ny*caveAreaSize + nx; !reached[j] && through(tiles[j]) {
				reached[j] = true
				queue = append(queue, j)
			}
		}
	}
	return reached
}

func isMountain(t data.TileEnum) bool {
	return t == data.Tile_Rock || t == data.Tile_RockFloor
}

// Only the middle of the area is judged, since what's near its edges may connect through tiles outside it
func inMiddle(i int) bool {
	x, y := i%caveAreaSize, i/caveAreaSize
	return x >= caveAreaSize/6 && x < caveAreaSize*5/6 && y >= caveAreaSize/6 && y < caveAreaSize*5/6
}

func TestCavesOpenOntoTheOutside(t *testing.T) {
	for _, seed := range []uint64{1, 7, 0x5EED} {
		tiles := caveArea(seed)
		outside := []int{}
		for i, tile := range tiles {
			if !isMountain(tile) {
				outside = append(outside, i)
			}
		}
		reached := flood(tiles, outside, func(t data.TileEnum) bool { return t != data.Tile_Rock })

		floor, reachedFloor := 0, 0
		for i, tile := range tiles {
			if tile == data.Tile_RockFloor && inMiddle(i) {
				floor++
				if reached[i] {
					reachedFloor++
				}
			}
		}
		if floor == 0 {
			t.Fatalf("seed %d: no caves", seed)
		}
		if share := float64(reachedFloor) / float64(floor); share < 0.95 {
			t.Errorf("seed %d: only %.2f of the cave floor can be walked to from outside", seed, share)
		}
	}
}

// Few, so that there's open land between them, but large, so that they have room for caves deep inside them
func TestMountainsAreFewButLarge(t *testing.T) {
	const depth = 8
	for _, seed := range []uint64{1, 7, 0x5EED} {
		tiles := caveArea(seed)
		mountain, deep := 0, 0
		for y := depth; y < caveAreaSize-depth; y++ {
			for x := depth; x < caveAreaSize-depth; x++ {
				if !isMountain(tiles[y*caveAreaSize+x]) {
					continue
				}
				mountain++
				if surroundedByMountain(tiles, x, y, depth) {
					deep++
				}
			}
		}

		if share := float64(mountain) / float64(len(tiles)); share < 0.05 || share > 0.2 {
			t.Errorf("seed %d: mountains cover %.2f of the land, want between 0.05 and 0.2", seed, share)
		}
		if share := float64(deep) / float64(mountain); share < 0.15 {
			t.Errorf("seed %d: only %.2f of the mountains are %d tiles from their edge, want at least 0.15", seed, share, depth)
		}
	}
}

func surroundedByMountain(tiles []data.TileEnum, x, y, depth int) bool {
	for dy := -depth; dy <= depth; dy++ {
		for dx := -depth; dx <= depth; dx++ {
			if !isMountain(tiles[(y+dy)*caveAreaSize+x+dx]) {
				return false
			}
		}
	}
	return true
}
