package shapes

import (
	"ebittest/utils"
	"math"
)

// The shape as it is on an entity turned by the rotation and scaled by the scale. A rectangle stays axis-aligned, with
// only its offset turned, so that it keeps its own collision checks.
func Transform(s Shape, rotation, scale float64) Shape {
	if rotation == 0 && scale == 1 {
		return s
	}

	place := func(v utils.Vec2f) utils.Vec2f {
		return v.Rotated(rotation).Multiply(scale)
	}

	switch x := s.(type) {
	case *CircleShape:
		c, _ := NewCircleShape(x.radius*scale, place(x.offset))
		return c
	case *RectangleShape:
		r, _ := NewRectangleShapeFromCoords(x.topLeft.Multiply(scale), x.bottomRight.Multiply(scale), place(x.offset))
		return r
	case *PolygonShape:
		verts := make([]utils.Vec2f, len(x.vertices))
		for i, v := range x.vertices {
			verts[i] = place(v)
		}
		p, _ := NewPolygonShape(verts, place(x.offset))
		return p
	}

	return s
}

func BoundingAABB(colShapes []Shape) [2]utils.Vec2f {
	if len(colShapes) == 0 {
		return [2]utils.Vec2f{}
	}

	bounds := colShapes[0].GetAABB()
	for _, shape := range colShapes[1:] {
		aabb := shape.GetAABB()
		bounds[0] = utils.Vec2f{X: math.Min(bounds[0].X, aabb[0].X), Y: math.Min(bounds[0].Y, aabb[0].Y)}
		bounds[1] = utils.Vec2f{X: math.Max(bounds[1].X, aabb[1].X), Y: math.Max(bounds[1].Y, aabb[1].Y)}
	}
	return bounds
}
