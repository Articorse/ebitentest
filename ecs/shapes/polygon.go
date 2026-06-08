package shapes

import (
	"ebittest/utils"
	"fmt"
	"math/rand/v2"
)

type PolygonShape struct {
	vertices []utils.Vec2
	offset   utils.Vec2
	aabb     [2]utils.Vec2
}

func (x *PolygonShape) Copy() Shape {
	verticesCopy := make([]utils.Vec2, len(x.vertices))
	copy(verticesCopy, x.vertices)

	return &PolygonShape{
		vertices: verticesCopy,
		offset:   x.offset,
		aabb:     x.aabb,
	}
}

func NewPolygonShape(v []utils.Vec2, o utils.Vec2) (*PolygonShape, error) {
	if len(v) < 3 {
		return nil, fmt.Errorf("a polygon shape must have at least 3 vertices")
	}

	var minX, minY, maxX, maxY float64
	minX, minY = v[0].X, v[0].Y
	maxX, maxY = v[0].X, v[0].Y

	for _, vertex := range v {
		if vertex.X < minX {
			minX = vertex.X
		}
		if vertex.Y < minY {
			minY = vertex.Y
		}
		if vertex.X > maxX {
			maxX = vertex.X
		}
		if vertex.Y > maxY {
			maxY = vertex.Y
		}
	}

	return &PolygonShape{
		vertices: v,
		offset:   o,
		aabb: [2]utils.Vec2{
			{X: minX + o.X, Y: minY + o.Y},
			{X: maxX + o.X, Y: maxY + o.Y},
		},
	}, nil
}

func (x *PolygonShape) GetVertices() []utils.Vec2 {
	return x.vertices
}

func (x *PolygonShape) GetOffset() utils.Vec2 {
	return x.offset
}

func (x *PolygonShape) GetAABB() [2]utils.Vec2 {
	return x.aabb
}

// TODO: Implement
func (x *PolygonShape) GetRandomPoint(r rand.Rand) utils.Vec2 {
	return utils.Vec2{}
}

// TODO: Implement
func (x *PolygonShape) GetRandomPointAroundShape(r rand.Rand) utils.Vec2 {
	return utils.Vec2{}
}
