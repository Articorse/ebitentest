package shapes

import (
	"ebittest/utils"
	"math"
)

// Whether the shape reaches into the box from lo to hi, both in the space the shape's offset is in. Strictly, so that a
// shape only touching the box's edge, such as something pressed flat against a wall, isn't in it.
func OverlapsBox(s Shape, lo, hi utils.Vec2f) bool {
	aabb := s.GetAABB()
	if aabb[0].X >= hi.X || aabb[1].X <= lo.X || aabb[0].Y >= hi.Y || aabb[1].Y <= lo.Y {
		return false
	}

	switch x := s.(type) {
	case *CircleShape:
		c := x.GetOffset()
		closest := utils.Vec2f{X: math.Max(lo.X, math.Min(c.X, hi.X)), Y: math.Max(lo.Y, math.Min(c.Y, hi.Y))}
		return closest.Subtract(c).Length() < x.GetRadius()
	case *PolygonShape:
		verts := make([]utils.Vec2f, len(x.vertices))
		for i, v := range x.vertices {
			verts[i] = v.Add(x.offset)
		}
		box := []utils.Vec2f{lo, {X: hi.X, Y: lo.Y}, hi, {X: lo.X, Y: hi.Y}}
		for i, a := range verts {
			b := verts[(i+1)%len(verts)]
			axis := utils.Vec2f{X: a.Y - b.Y, Y: b.X - a.X}
			pMin, pMax := project(verts, axis)
			bMin, bMax := project(box, axis)
			if pMin >= bMax || bMin >= pMax {
				return false
			}
		}
		return true
	}
	return true
}

func project(verts []utils.Vec2f, axis utils.Vec2f) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, v := range verts {
		d := v.Dot(axis)
		lo, hi = math.Min(lo, d), math.Max(hi, d)
	}
	return lo, hi
}
