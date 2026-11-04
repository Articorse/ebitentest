package pathfinding_test

import (
	"ebittest/pathfinding"
	"ebittest/utils"
	"math/rand/v2"
	"slices"
	"testing"
)

// Walkable at full speed within 30 tiles of the origin, except where told
type grid struct {
	walls map[utils.Vec2i]bool
	slow  map[utils.Vec2i]int
}

func newGrid(walls ...utils.Vec2i) grid {
	g := grid{walls: map[utils.Vec2i]bool{}, slow: map[utils.Vec2i]int{}}
	for _, w := range walls {
		g.walls[w] = true
	}
	return g
}

func (g grid) speed(t utils.Vec2i) int {
	if utils.AbsInt(t.X) > 30 || utils.AbsInt(t.Y) > 30 || g.walls[t] {
		return 0
	}
	if s, ok := g.slow[t]; ok {
		return s
	}
	return 100
}

// What walking the path from from costs, the same way the search counts it
func cost(t *testing.T, g grid, from utils.Vec2i, path []utils.Vec2i) int {
	t.Helper()
	total, at := 0, from
	for _, next := range path {
		dx, dy := utils.AbsInt(next.X-at.X), utils.AbsInt(next.Y-at.Y)
		if dx > 1 || dy > 1 || dx+dy == 0 {
			t.Fatalf("the path steps from %v to %v", at, next)
		}
		step := 100
		if dx+dy == 2 {
			if g.speed(utils.Vec2i{X: next.X, Y: at.Y}) == 0 || g.speed(utils.Vec2i{X: at.X, Y: next.Y}) == 0 {
				t.Fatalf("the path cuts the corner from %v to %v", at, next)
			}
			step = 141
		}
		s := g.speed(next)
		if s == 0 {
			t.Fatalf("the path walks into %v, which can't be walked on", next)
		}
		total += step * 100 / s
		at = next
	}
	return total
}

func TestAPathOverOpenGroundGoesStraightAndDiagonally(t *testing.T) {
	g := newGrid()
	path, ok := pathfinding.Find(utils.Vec2i{}, utils.Vec2i{X: 5, Y: 2}, 1000, g.speed)
	if !ok || len(path) != 5 || path[len(path)-1] != (utils.Vec2i{X: 5, Y: 2}) {
		t.Fatalf("over open ground to (5, 2), the path is %v (%v), want 5 steps", path, ok)
	}
	if got := cost(t, g, utils.Vec2i{}, path); got != 3*100+2*141 {
		t.Fatalf("the path costs %d, want 3 straight and 2 diagonal steps", got)
	}

	if path, ok := pathfinding.Find(utils.Vec2i{X: 2}, utils.Vec2i{X: 2}, 1000, g.speed); !ok || len(path) != 0 {
		t.Fatalf("to its own tile, the path is %v (%v), want empty", path, ok)
	}
}

func TestPathsGoAroundWallsWithoutCuttingCorners(t *testing.T) {
	g := newGrid(utils.Vec2i{X: 1}, utils.Vec2i{X: 1, Y: 1}, utils.Vec2i{X: 1, Y: -1}, utils.Vec2i{X: 1, Y: 2}, utils.Vec2i{X: 1, Y: -2})
	path, ok := pathfinding.Find(utils.Vec2i{}, utils.Vec2i{X: 2}, 1000, g.speed)
	if !ok {
		t.Fatal("no path around a wall")
	}
	cost(t, g, utils.Vec2i{}, path)

	box := newGrid()
	for x := -2; x <= 2; x++ {
		for y := -2; y <= 2; y++ {
			if (utils.AbsInt(x) == 2 || utils.AbsInt(y) == 2) && !(x == 2 && y == 2) {
				box.walls[utils.Vec2i{X: x, Y: y}] = true
			}
		}
	}
	// The box's only way out is its corner tile, which can only be reached diagonally between two walls
	if path, ok := pathfinding.Find(utils.Vec2i{}, utils.Vec2i{X: 5, Y: 5}, 10000, box.speed); ok {
		t.Fatalf("the path squeezes diagonally between two walls: %v", path)
	}
}

func TestSlowTilesAreCrossedOnlyWhenGoingAroundCostsMore(t *testing.T) {
	g := newGrid()
	for y := -30; y <= 30; y++ {
		g.slow[utils.Vec2i{X: 1, Y: y}] = 70
	}
	path, ok := pathfinding.Find(utils.Vec2i{}, utils.Vec2i{X: 2}, 1000, g.speed)
	if !ok || !slices.Contains(path, utils.Vec2i{X: 1}) {
		t.Fatalf("with a band of slow tiles in the way, the path is %v, want it straight through", path)
	}

	pond := newGrid()
	pond.slow[utils.Vec2i{X: 1}] = 30
	path, ok = pathfinding.Find(utils.Vec2i{}, utils.Vec2i{X: 2}, 1000, pond.speed)
	if !ok || slices.Contains(path, utils.Vec2i{X: 1}) {
		t.Fatalf("with one slow tile in the way, the path is %v, want it around", path)
	}
}

func TestNoPathWhenTheGoalIsWalledOffOrTooFar(t *testing.T) {
	ring := newGrid()
	for x := -1; x <= 1; x++ {
		for y := -1; y <= 1; y++ {
			if x != 0 || y != 0 {
				ring.walls[utils.Vec2i{X: 10 + x, Y: y}] = true
			}
		}
	}
	if path, ok := pathfinding.Find(utils.Vec2i{}, utils.Vec2i{X: 10}, 100000, ring.speed); ok {
		t.Fatalf("found a path %v into a walled off tile", path)
	}
	if _, ok := pathfinding.Find(utils.Vec2i{}, utils.Vec2i{X: 20}, 10, newGrid().speed); ok {
		t.Fatal("found a path 20 tiles away after looking at 10 tiles")
	}
}

func TestTheStartAndGoalCanBeTilesThatCantBeWalked(t *testing.T) {
	g := newGrid(utils.Vec2i{}, utils.Vec2i{X: 3})
	path, ok := pathfinding.Find(utils.Vec2i{}, utils.Vec2i{X: 3}, 1000, g.speed)
	if !ok || len(path) != 3 {
		t.Fatalf("from a tree's tile to another, the path is %v (%v), want 3 steps", path, ok)
	}
}

// Against a plain Dijkstra search, over random grids
func TestPathsAreTheCheapestAndTheSameEveryTime(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range 15 {
		g := newGrid()
		for range 400 {
			tile := utils.Vec2i{X: rng.IntN(41) - 20, Y: rng.IntN(41) - 20}
			if rng.IntN(3) == 0 {
				g.slow[tile] = 30 + rng.IntN(70)
			} else {
				g.walls[tile] = true
			}
		}
		from, to := utils.Vec2i{X: -15, Y: -15}, utils.Vec2i{X: 15, Y: 15}
		delete(g.walls, from)
		delete(g.walls, to)

		want, reachable := dijkstra(g, from, to)
		path, ok := pathfinding.Find(from, to, 100000, g.speed)
		if ok != reachable {
			t.Fatalf("found a path %v, want %v", ok, reachable)
		}
		if !ok {
			continue
		}
		if got := cost(t, g, from, path); got != want {
			t.Fatalf("the path costs %d, want the cheapest, %d", got, want)
		}
		again, _ := pathfinding.Find(from, to, 100000, g.speed)
		if !slices.Equal(path, again) {
			t.Fatal("the same search found a different path")
		}
	}
}

func dijkstra(g grid, from, to utils.Vec2i) (int, bool) {
	best := map[utils.Vec2i]int{from: 0}
	done := map[utils.Vec2i]bool{}
	for {
		cur, curCost, found := utils.Vec2i{}, 0, false
		for tile, c := range best {
			if !done[tile] && (!found || c < curCost) {
				cur, curCost, found = tile, c, true
			}
		}
		if !found {
			return 0, false
		}
		if cur == to {
			return curCost, true
		}
		done[cur] = true
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				next := utils.Vec2i{X: cur.X + dx, Y: cur.Y + dy}
				speed := g.speed(next)
				if (dx == 0 && dy == 0) || speed == 0 {
					continue
				}
				step := 100
				if dx != 0 && dy != 0 {
					if g.speed(utils.Vec2i{X: next.X, Y: cur.Y}) == 0 || g.speed(utils.Vec2i{X: cur.X, Y: next.Y}) == 0 {
						continue
					}
					step = 141
				}
				c := curCost + step*100/speed
				if old, ok := best[next]; !ok || c < old {
					best[next] = c
				}
			}
		}
	}
}
