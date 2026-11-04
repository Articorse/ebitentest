// A* over the tile grid in eight directions, in integers and with every tie broken the same way, so that the same
// query always finds the same path.
package pathfinding

import (
	"container/heap"
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
	if from == to {
		return nil, true
	}

	speed := func(t utils.Vec2i) int {
		if t == from || t == to {
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
	heap.Push(open, entry{tile: from, f: heuristic(from, to), h: heuristic(from, to), seq: seq})

	for expanded := 0; open.Len() > 0 && expanded < maxExpanded; {
		current := heap.Pop(open).(entry)
		v := visits[current.tile]
		if v.closed {
			continue
		}
		v.closed = true
		expanded++

		if current.tile == to {
			path := []utils.Vec2i{}
			for t := to; t != from; t = visits[t].parent {
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
			h := heuristic(next, to)
			heap.Push(open, entry{tile: next, f: cost + h, h: h, seq: seq})
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

// Cheapest first, then closest to the goal, then the earliest found
type queue []entry

func (q queue) Len() int { return len(q) }
func (q queue) Less(i, j int) bool {
	if q[i].f != q[j].f {
		return q[i].f < q[j].f
	}
	if q[i].h != q[j].h {
		return q[i].h < q[j].h
	}
	return q[i].seq < q[j].seq
}
func (q queue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *queue) Push(x any)   { *q = append(*q, x.(entry)) }
func (q *queue) Pop() any {
	old := *q
	e := old[len(old)-1]
	*q = old[:len(old)-1]
	return e
}
