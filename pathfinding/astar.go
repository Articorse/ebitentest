// A*, and Dijkstra for the nearest of many goals, over the tile grid in eight directions, in integers and with every tie
// broken the same way, so that the same query always finds the same path.
package pathfinding

import (
	"ebittest/utils"
)

const (
	// What a step onto a tile walked at full speed costs; slower tiles cost more in proportion
	straightStepCost = 100
	// √2 times a straight step
	diagonalStepCost = 141
)

// In a fixed order, straight steps first
var directions = [8]utils.Vec2i{
	{X: 1}, {X: -1}, {Y: 1}, {Y: -1},
	{X: 1, Y: 1}, {X: -1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: -1},
}

// The cheapest path from one tile to another, walking at speedPercent (0 where a tile can't be walked on) and giving up
// after expanding maxExpanded tiles. It runs from the tile after from to to. The start and goal tiles count as walkable
// whatever they are, since something can stand partly on a tile it couldn't walk through, such as beside a tree trunk.
// A diagonal step needs both tiles beside it walkable, so that paths don't cut the corners of walls.
func Find(from, to utils.Vec2i, maxExpanded int, speedPercent func(utils.Vec2i) int) ([]utils.Vec2i, bool) {
	speed := func(t utils.Vec2i) int {
		if t == to {
			return 100
		}
		return speedPercent(t)
	}
	isGoal := func(t utils.Vec2i) bool { return t == to }
	return search(from, isGoal, func(t utils.Vec2i) int { return heuristic(t, to) }, maxExpanded, speed)
}

func search(
	from utils.Vec2i,
	isGoal func(utils.Vec2i) bool,
	h func(utils.Vec2i) int,
	maxExpanded int,
	speedPercent func(utils.Vec2i) int,
) ([]utils.Vec2i, bool) {
	if isGoal(from) {
		return nil, true
	}

	speed := func(t utils.Vec2i) int {
		if t == from {
			return 100
		}
		return speedPercent(t)
	}

	type visit struct {
		cost   int
		parent utils.Vec2i
		closed bool
	}
	visits := map[utils.Vec2i]*visit{from: {}}
	open := &queue{}
	seq := 0
	open.push(entry{tile: from, f: h(from), h: h(from), seq: seq})

	for expanded := 0; len(*open) > 0 && expanded < maxExpanded; {
		current := open.pop()
		v := visits[current.tile]
		if v.closed {
			continue
		}
		v.closed = true
		expanded++

		if isGoal(current.tile) {
			path := []utils.Vec2i{}
			for t := current.tile; t != from; t = visits[t].parent {
				path = append(path, t)
			}
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}
			return path, true
		}

		for i, d := range directions {
			next := utils.Vec2i{X: current.tile.X + d.X, Y: current.tile.Y + d.Y}
			s := speed(next)
			if s <= 0 {
				continue
			}
			step := straightStepCost
			if i >= 4 {
				if speed(utils.Vec2i{X: next.X, Y: current.tile.Y}) <= 0 || speed(utils.Vec2i{X: current.tile.X, Y: next.Y}) <= 0 {
					continue
				}
				step = diagonalStepCost
			}

			cost := v.cost + step*100/s
			if nv, ok := visits[next]; ok && (nv.closed || nv.cost <= cost) {
				continue
			}
			visits[next] = &visit{cost: cost, parent: current.tile}
			seq++
			nh := h(next)
			open.push(entry{tile: next, f: cost + nh, h: nh, seq: seq})
		}
	}
	return nil, false
}

// The cost of the shortest path over open ground, which no path can beat
func heuristic(a, b utils.Vec2i) int {
	dx, dy := utils.AbsInt(a.X-b.X), utils.AbsInt(a.Y-b.Y)
	return straightStepCost*max(dx, dy) + (diagonalStepCost-straightStepCost)*min(dx, dy)
}

type entry struct {
	tile utils.Vec2i
	f, h int
	seq  int
}

// Cheapest first, then closest to the goal, then the earliest found. A binary heap of its own rather than
// container/heap's, which boxes every entry pushed and popped, since a search pushes and pops one for every tile.
type queue []entry

func (q queue) less(i, j int) bool {
	if q[i].f != q[j].f {
		return q[i].f < q[j].f
	}
	if q[i].h != q[j].h {
		return q[i].h < q[j].h
	}
	return q[i].seq < q[j].seq
}

func (q *queue) push(e entry) {
	*q = append(*q, e)
	for i := len(*q) - 1; i > 0; {
		parent := (i - 1) / 2
		if !q.less(i, parent) {
			break
		}
		(*q)[i], (*q)[parent] = (*q)[parent], (*q)[i]
		i = parent
	}
}

func (q *queue) pop() entry {
	old := *q
	top := old[0]
	last := len(old) - 1
	old[0] = old[last]
	*q = old[:last]
	for i := 0; ; {
		smallest, l, r := i, 2*i+1, 2*i+2
		if l < last && q.less(l, smallest) {
			smallest = l
		}
		if r < last && q.less(r, smallest) {
			smallest = r
		}
		if smallest == i {
			break
		}
		(*q)[i], (*q)[smallest] = (*q)[smallest], (*q)[i]
		i = smallest
	}
	return top
}
