// Symmetric shadowcasting, after Albert Ford's description: https://www.albertford.com/shadowcasting/
// Slopes are kept as fractions, so that the result is exact and the same everywhere.
package fov

import "ebittest/utils"

// Marks every tile within radius that can be seen from origin. Opaque tiles are visible themselves, so walls show,
// but block what's behind them. If a can see b, b can see a. A tile may be marked more than once.
func Compute(origin utils.Vec2i, radius int, isOpaque func(utils.Vec2i) bool, markVisible func(utils.Vec2i)) {
	markVisible(origin)

	for _, q := range quadrants {
		s := scanner{origin: origin, quadrant: q, radius: radius, isOpaque: isOpaque, markVisible: markVisible}
		s.scan(row{depth: 1, start: fraction{-1, 1}, end: fraction{1, 1}})
	}
}

type quadrant int

const (
	north quadrant = iota
	east
	south
	west
)

var quadrants = [...]quadrant{north, east, south, west}

type fraction struct {
	num, den int
}

type row struct {
	depth      int
	start, end fraction
}

type scanner struct {
	origin      utils.Vec2i
	quadrant    quadrant
	radius      int
	isOpaque    func(utils.Vec2i) bool
	markVisible func(utils.Vec2i)
}

func (s *scanner) scan(r row) {
	if r.depth > s.radius {
		return
	}

	// Ties round towards the middle of the row, so that a tile only shows when its center is inside the row's range
	minCol := floorDiv(2*r.depth*r.start.num+r.start.den, 2*r.start.den)
	maxCol := ceilDiv(2*r.depth*r.end.num-r.end.den, 2*r.end.den)

	prevWall, hasPrev := false, false
	for col := minCol; col <= maxCol; col++ {
		tile := s.toWorld(r.depth, col)
		wall := s.isOpaque(tile)

		if (wall || isSymmetric(r, col)) && r.depth*r.depth+col*col <= s.radius*s.radius {
			s.markVisible(tile)
		}

		if hasPrev && prevWall && !wall {
			r.start = slope(r.depth, col)
		}
		if hasPrev && !prevWall && wall {
			next := row{depth: r.depth + 1, start: r.start, end: slope(r.depth, col)}
			s.scan(next)
		}

		prevWall, hasPrev = wall, true
	}

	if hasPrev && !prevWall {
		s.scan(row{depth: r.depth + 1, start: r.start, end: r.end})
	}
}

func (s *scanner) toWorld(depth, col int) utils.Vec2i {
	switch s.quadrant {
	case north:
		return utils.Vec2i{X: s.origin.X + col, Y: s.origin.Y - depth}
	case south:
		return utils.Vec2i{X: s.origin.X + col, Y: s.origin.Y + depth}
	case east:
		return utils.Vec2i{X: s.origin.X + depth, Y: s.origin.Y + col}
	default:
		return utils.Vec2i{X: s.origin.X - depth, Y: s.origin.Y + col}
	}
}

// The slope of the tile's left edge, as seen from the origin
func slope(depth, col int) fraction {
	return fraction{2*col - 1, 2 * depth}
}

// Only floor tiles whose center lies inside the row's range are visible, which is what makes visibility symmetric
func isSymmetric(r row, col int) bool {
	return col*r.start.den >= r.depth*r.start.num && col*r.end.den <= r.depth*r.end.num
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func ceilDiv(a, b int) int {
	return -floorDiv(-a, b)
}
