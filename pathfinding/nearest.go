package pathfinding

import (
	"ebittest/utils"
)

// The cheapest path to the closest tile that isGoal, by walking cost rather than distance, searching only the tiles
// from first to last. It's Find without a heuristic, so it spreads out evenly until it reaches a goal, and since that
// can mean searching all of a large area, it keeps what it knows about each tile in arrays rather than maps, and asks
// for each tile's speed once. Only the start counts as walkable whatever it is, since a goal is somewhere to stand.
// Nil and true if the start is a goal.
func FindNearest(
	from, first, last utils.Vec2i,
	isGoal func(utils.Vec2i) bool,
	maxExpanded int,
	speedPercent func(utils.Vec2i) int,
) ([]utils.Vec2i, bool) {
	if isGoal(from) {
		return nil, true
	}
	w, h := last.X-first.X+1, last.Y-first.Y+1
	index := func(t utils.Vec2i) int {
		if t.X < first.X || t.X > last.X || t.Y < first.Y || t.Y > last.Y {
			return -1
		}
		return (t.Y-first.Y)*w + t.X - first.X
	}
	start := index(from)
	if start == -1 {
		return nil, false
	}

	// -1 until asked
	speeds := make([]int16, w*h)
	costs := make([]int32, w*h)
	parents := make([]int32, w*h)
	closed := make([]bool, w*h)
	for i := range speeds {
		speeds[i], costs[i] = -1, -1
	}
	speed := func(t utils.Vec2i) int {
		i := index(t)
		if i == -1 {
			return 0
		}
		if i == start {
			return 100
		}
		if speeds[i] == -1 {
			speeds[i] = int16(speedPercent(t))
		}
		return int(speeds[i])
	}

	costs[start] = 0
	open := &queue{}
	seq := 0
	open.push(entry{tile: from, seq: seq})

	for expanded := 0; len(*open) > 0 && expanded < maxExpanded; {
		current := open.pop()
		ci := index(current.tile)
		if closed[ci] {
			continue
		}
		closed[ci] = true
		expanded++

		if isGoal(current.tile) {
			path := []utils.Vec2i{}
			for i := ci; i != start; i = int(parents[i]) {
				path = append(path, utils.Vec2i{X: first.X + i%w, Y: first.Y + i/w})
			}
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}
			return path, true
		}

		for d, dir := range directions {
			next := utils.Vec2i{X: current.tile.X + dir.X, Y: current.tile.Y + dir.Y}
			s := speed(next)
			if s <= 0 {
				continue
			}
			step := straightStepCost
			if d >= 4 {
				if speed(utils.Vec2i{X: next.X, Y: current.tile.Y}) <= 0 || speed(utils.Vec2i{X: current.tile.X, Y: next.Y}) <= 0 {
					continue
				}
				step = diagonalStepCost
			}

			ni := index(next)
			cost := int(costs[ci]) + step*100/s
			if closed[ni] || (costs[ni] != -1 && int(costs[ni]) <= cost) {
				continue
			}
			costs[ni], parents[ni] = int32(cost), int32(ci)
			seq++
			open.push(entry{tile: next, f: cost, seq: seq})
		}
	}
	return nil, false
}
