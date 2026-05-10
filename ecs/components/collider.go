package components

import (
	"ebittest/utils"
	"fmt"
	"math"
)

type Collider struct {
	Vertices []utils.Vec2
	AABB     []utils.Vec2
}

func NewColliderComponent(v []utils.Vec2) (*Collider, error) {
	if !isSimple(v) {
		return nil, fmt.Errorf("collider has self-intersections")
	}

	if !isConvex(v) {
		return nil, fmt.Errorf("collider must be convex")
	}

	aabb := generateAABB(v)

	return &Collider{Vertices: v, AABB: aabb}, nil
}

func generateAABB(vertices []utils.Vec2) []utils.Vec2 {
	maxX := math.Inf(-1)
	maxY := math.Inf(-1)
	minX := math.Inf(0)
	minY := math.Inf(0)

	for _, v := range vertices {
		if v.X > maxX {
			maxX = v.X
		}
		if v.X < minX {
			minX = v.X
		}
		if v.Y > maxY {
			maxY = v.Y
		}
		if v.Y < minY {
			minY = v.Y
		}
	}

	return []utils.Vec2{
		{X: minX, Y: minY},
		{X: maxX, Y: maxY},
	}
}

func isConvex(vertices []utils.Vec2) bool {
	n := len(vertices)
	if n < 3 {
		return false
	}
	var sign float64
	for i := 0; i < n; i++ {
		a := vertices[i]
		b := vertices[(i+1)%n]
		c := vertices[(i+2)%n]
		cross := (b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)
		if cross != 0 {
			if sign == 0 {
				sign = cross
			} else if sign*cross < 0 {
				return false
			}
		}
	}
	return true
}

func isSimple(vertices []utils.Vec2) bool {
	n := len(vertices)
	for i := 0; i < n; i++ {
		a1 := vertices[i]
		a2 := vertices[(i+1)%n]
		for j := i + 1; j < n; j++ {
			// Skip adjacent edges and the same edge
			if utils.AbsInt(i-j) <= 1 || (i == 0 && j == n-1) || (j == 0 && i == n-1) {
				continue
			}
			b1 := vertices[j]
			b2 := vertices[(j+1)%n]
			if utils.SegmentsIntersect(a1, a2, b1, b2) {
				return false
			}
		}
	}
	return true
}
