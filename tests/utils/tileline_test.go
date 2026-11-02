package utils_test

import (
	"ebittest/data"
	"ebittest/utils"
	"slices"
	"testing"
)

func TestTilesAlongSegment(t *testing.T) {
	const s = data.TileSize
	cases := []struct {
		name string
		a, b utils.Vec2f
		want []utils.Vec2i
	}{
		{"within one tile", utils.Vec2f{X: 1, Y: 2}, utils.Vec2f{X: -3, Y: 4}, []utils.Vec2i{{}}},
		{"straight right", utils.Vec2f{}, utils.Vec2f{X: 3 * s}, []utils.Vec2i{{}, {X: 1}, {X: 2}, {X: 3}}},
		{"straight up", utils.Vec2f{}, utils.Vec2f{Y: -2 * s}, []utils.Vec2i{{}, {Y: -1}, {Y: -2}}},
		{"shallow", utils.Vec2f{}, utils.Vec2f{X: 3 * s, Y: 0.8 * s}, []utils.Vec2i{{}, {X: 1}, {X: 2}, {X: 2, Y: 1}, {X: 3, Y: 1}}},
		{"steep", utils.Vec2f{}, utils.Vec2f{X: 0.8 * s, Y: -3 * s}, []utils.Vec2i{{}, {Y: -1}, {Y: -2}, {X: 1, Y: -2}, {X: 1, Y: -3}}},
		{"through a corner", utils.Vec2f{}, utils.Vec2f{X: -s, Y: s}, []utils.Vec2i{{}, {X: -1}, {X: -1, Y: 1}}},
		{"from off-center", utils.Vec2f{X: 7, Y: 7}, utils.Vec2f{X: 9, Y: 9}, []utils.Vec2i{{}, {X: 1}, {X: 1, Y: 1}}},
	}

	for _, c := range cases {
		if got := utils.TilesAlongSegment(c.a, c.b); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestTilesAlongSegmentAreConnectedAndEndAtTheEnd(t *testing.T) {
	for i := range 200 {
		a := utils.Vec2f{X: float64(i*37%301) - 150, Y: float64(i*53%299) - 150}
		b := utils.Vec2f{X: float64(i*71%307) - 150, Y: float64(i*13%293) - 150}
		tiles := utils.TilesAlongSegment(a, b)
		if tiles[0] != utils.WorldPosToTilePos(a) || tiles[len(tiles)-1] != utils.WorldPosToTilePos(b) {
			t.Fatalf("%v to %v: tiles %v don't run from start to end", a, b, tiles)
		}
		for j := 1; j < len(tiles); j++ {
			if d := utils.Dist2(tiles[j-1], tiles[j]); d != 1 {
				t.Fatalf("%v to %v: tiles %v and %v aren't side by side", a, b, tiles[j-1], tiles[j])
			}
		}
	}
}
