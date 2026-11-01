package fov_test

import (
	"ebittest/fov"
	"ebittest/utils"
	"testing"
)

type grid map[utils.Vec2i]bool

func (g grid) opaque(t utils.Vec2i) bool { return g[t] }

func visibleFrom(origin utils.Vec2i, radius int, g grid) map[utils.Vec2i]bool {
	seen := map[utils.Vec2i]bool{}
	fov.Compute(origin, radius, g.opaque, func(t utils.Vec2i) { seen[t] = true })
	return seen
}

func TestOpenGroundShowsExactlyTheCircle(t *testing.T) {
	const r = 7
	origin := utils.Vec2i{X: 3, Y: -2}
	seen := visibleFrom(origin, r, grid{})

	for y := -r - 1; y <= r+1; y++ {
		for x := -r - 1; x <= r+1; x++ {
			tile := utils.Vec2i{X: origin.X + x, Y: origin.Y + y}
			if want := x*x+y*y <= r*r; seen[tile] != want {
				t.Fatalf("tile %v visible %v, want %v", tile, seen[tile], want)
			}
		}
	}
}

func TestWallBlocksWhatsBehindItButShowsItself(t *testing.T) {
	g := grid{}
	for y := -4; y <= 4; y++ {
		g[utils.Vec2i{X: 3, Y: y}] = true
	}
	seen := visibleFrom(utils.Vec2i{}, 10, g)

	for y := -1; y <= 1; y++ {
		if !seen[utils.Vec2i{X: 3, Y: y}] {
			t.Errorf("wall tile (3, %d) isn't visible", y)
		}
		for x := 4; x <= 9; x++ {
			if seen[utils.Vec2i{X: x, Y: y}] {
				t.Errorf("tile (%d, %d) behind the wall is visible", x, y)
			}
		}
	}
	if !seen[utils.Vec2i{X: -5}] || !seen[utils.Vec2i{Y: 6}] {
		t.Error("open tiles away from the wall aren't visible")
	}
}

func TestOriginIsVisibleEvenInsideAWall(t *testing.T) {
	if seen := visibleFrom(utils.Vec2i{}, 3, grid{{}: true}); !seen[utils.Vec2i{}] {
		t.Fatal("origin isn't visible")
	}
}

func TestVisibilityIsSymmetric(t *testing.T) {
	const r = 9
	g := grid{}
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if (x*7919+y*104729)%10 < 3 && (x != 0 || y != 0) {
				g[utils.Vec2i{X: x, Y: y}] = true
			}
		}
	}

	floors := []utils.Vec2i{}
	for y := -r / 2; y <= r/2; y++ {
		for x := -r / 2; x <= r/2; x++ {
			if !g[utils.Vec2i{X: x, Y: y}] {
				floors = append(floors, utils.Vec2i{X: x, Y: y})
			}
		}
	}

	views := map[utils.Vec2i]map[utils.Vec2i]bool{}
	for _, f := range floors {
		views[f] = visibleFrom(f, r, g)
	}
	for _, a := range floors {
		for _, b := range floors {
			if views[a][b] != views[b][a] {
				t.Fatalf("%v sees %v is %v, but %v sees %v is %v", a, b, views[a][b], b, a, views[b][a])
			}
		}
	}
}
