package shapes

import (
	"ebittest/utils"
	"math"
	"math/rand/v2"
)

type triangle struct {
	v1, v2, v3 utils.Vec2f
	area       float64
}

// PolygonShape represents a convex polygon defined by its vertices.
type PolygonShape struct {
	vertices []utils.Vec2f
	offset   utils.Vec2f
	aabb     [2]utils.Vec2f

	cachedTriangles []triangle
	cachedTotalArea float64
}

func NewPolygonShape(v []utils.Vec2f, o utils.Vec2f) (*PolygonShape, error) {
	if len(v) < 3 {
		return nil, nil
	}

	minX, minY := v[0].X, v[0].Y
	maxX, maxY := v[0].X, v[0].Y
	for _, vert := range v {
		minX = math.Min(minX, vert.X)
		minY = math.Min(minY, vert.Y)
		maxX = math.Max(maxX, vert.X)
		maxY = math.Max(maxY, vert.Y)
	}

	aabb := [2]utils.Vec2f{{X: minX + o.X, Y: minY + o.Y}, {X: maxX + o.X, Y: maxY + o.Y}}

	p := &PolygonShape{
		vertices: v,
		offset:   o,
		aabb:     aabb,
	}

	p.cachedTriangles = make([]triangle, 0, len(p.vertices)-2)
	p.cachedTotalArea = 0.0

	v0 := p.vertices[0]
	for i := 1; i < len(p.vertices)-1; i++ {
		v1 := p.vertices[i]
		v2 := p.vertices[i+1]

		currentTriangle := triangle{v1: v0, v2: v1, v3: v2}

		area := 0.5 * math.Abs(
			v0.X*(v1.Y-v2.Y)+
				v1.X*(v2.Y-v0.Y)+
				v2.X*(v0.Y-v1.Y),
		)
		currentTriangle.area = area
		p.cachedTotalArea += area
		p.cachedTriangles = append(p.cachedTriangles, currentTriangle)
	}

	return p, nil
}

func (x *PolygonShape) Copy() Shape {
	verticesCopy := make([]utils.Vec2f, len(x.vertices))
	copy(verticesCopy, x.vertices)

	cachedTrianglesCopy := make([]triangle, len(x.cachedTriangles))
	copy(cachedTrianglesCopy, x.cachedTriangles)

	return &PolygonShape{
		vertices:        verticesCopy,
		offset:          x.offset,
		aabb:            x.aabb,
		cachedTriangles: cachedTrianglesCopy,
		cachedTotalArea: x.cachedTotalArea,
	}
}

func (x *PolygonShape) GetVertices() []utils.Vec2f {
	return x.vertices
}

func (x *PolygonShape) GetOffset() utils.Vec2f {
	return x.offset
}

func (x *PolygonShape) GetAABB() [2]utils.Vec2f {
	return x.aabb
}

// This implementation uses barycentric coordinates within a randomly selected triangle
// to ensure uniform distribution within the convex polygon.
func (x *PolygonShape) GetRandomPoint(r *rand.Rand) utils.Vec2f {
	if len(x.vertices) < 3 || x.cachedTotalArea == 0 {
		return utils.Vec2f{} // Cannot generate a random point for degenerate polygon or zero area
	}

	// Randomly select a triangle weighted by area
	randArea := r.Float64() * x.cachedTotalArea
	selectedTriangle := triangle{}
	currentSumArea := 0.0
	for _, t := range x.cachedTriangles {
		currentSumArea += t.area
		if randArea <= currentSumArea {
			selectedTriangle = t
			break
		}
	}

	// Fallback if no triangle was selected (shouldn't happen with correct logic)
	if selectedTriangle.area == 0 {
		return utils.Vec2f{}
	}

	s := r.Float64()
	t := r.Float64()

	if s+t > 1 {
		s = 1 - s
		t = 1 - t
	}

	// Calculate vectors from v1 to v2 and v1 to v3
	vec12 := selectedTriangle.v2.Subtract(selectedTriangle.v1)
	vec13 := selectedTriangle.v3.Subtract(selectedTriangle.v1)

	// Calculate point using barycentric coordinates (relative to v1)
	randomPoint := selectedTriangle.v1.Add(vec12.Multiply(s)).Add(vec13.Multiply(t))

	// Adjust for the polygon's offset
	return randomPoint.Add(x.offset)
}

func (x *PolygonShape) GetRandomPointAroundShape(r *rand.Rand) utils.Vec2f {
	if len(x.vertices) < 2 {
		return utils.Vec2f{} // Cannot generate a random point for degenerate polygon
	}

	// Calculate the total perimeter length
	perimeter := 0.0
	for i := 0; i < len(x.vertices); i++ {
		v1 := x.vertices[i]
		v2 := x.vertices[(i+1)%len(x.vertices)]
		perimeter += v1.Subtract(v2).Length()
	}

	if perimeter == 0 {
		return utils.Vec2f{} // Cannot generate a random point for zero perimeter
	}

	// Randomly select a point along the perimeter
	randPerimeter := r.Float64() * perimeter
	currentLength := 0.0

	for i := 0; i < len(x.vertices); i++ {
		v1 := x.vertices[i]
		v2 := x.vertices[(i+1)%len(x.vertices)]
		edgeLength := v1.Subtract(v2).Length()

		if currentLength+edgeLength >= randPerimeter {
			t := (randPerimeter - currentLength) / edgeLength
			randomPoint := v1.Multiply(1 - t).Add(v2.Multiply(t))
			return randomPoint.Add(x.offset)
		}

		currentLength += edgeLength
	}

	return utils.Vec2f{} // Fallback, should not reach here
}

type PolygonParams struct {
	Vertices []utils.Vec2f
	Offset   utils.Vec2f
}

func (PolygonParams) isShapeParams() {}
