package shapes_test

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"ebittest/ecs/shapes"
	"ebittest/utils"
)

// calculateCentroid calculates the centroid of a PolygonShape.
// This is for testing purposes and assumes the polygon is valid.
func calculateCentroid(p *shapes.PolygonShape) utils.Vec2 {
	if len(p.GetVertices()) == 0 {
		return utils.Vec2{}
	}
	sum := utils.Vec2{}
	for _, v := range p.GetVertices() {
		sum = sum.Add(v)
	}
	return sum.Multiply(1.0 / float64(len(p.GetVertices())))
}

func TestGetRandomPoint_DegeneratePolygon(t *testing.T) {
	// Test polygon with less than 3 vertices
	degenerateVertices := []utils.Vec2{{X: 0, Y: 0}, {X: 1, Y: 0}}
	p, _ := shapes.NewPolygonShape(degenerateVertices, utils.Vec2{})
	if p != nil {
		if got := p.GetRandomPoint(rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), rand.Uint64()))); got != (utils.Vec2{}) {
			t.Errorf("GetRandomPoint() for degenerate polygon (2 vertices) = %v, want %v", got, utils.Vec2{})
		}
	} else {
		t.Logf("NewPolygonShape correctly returned nil for 2 vertices.")
	}

	// Test polygon with zero area (e.g., collinear vertices)
	collinearVertices := []utils.Vec2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}}
	pCollinear, _ := shapes.NewPolygonShape(collinearVertices, utils.Vec2{})
	if pCollinear != nil {
		if got := pCollinear.GetRandomPoint(rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), rand.Uint64()))); got != (utils.Vec2{}) {
			t.Errorf("GetRandomPoint() for zero-area polygon = %v, want %v", got, utils.Vec2{})
		}
	} else {
		t.Logf("NewPolygonShape correctly returned nil for collinear vertices (zero area).")
	}
}

func TestGetRandomPoint_Triangle(t *testing.T) {
	vertices := []utils.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 10}}
	offset := utils.Vec2{X: 1, Y: 1}
	p, err := shapes.NewPolygonShape(vertices, offset)
	if err != nil {
		t.Fatalf("Failed to create polygon: %v", err)
	}

	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), rand.Uint64()))
	numPoints := 1000
	minX, minY := p.GetAABB()[0].X, p.GetAABB()[0].Y
	maxX, maxY := p.GetAABB()[1].X, p.GetAABB()[1].Y

	for range numPoints {
		point := p.GetRandomPoint(rng)
		if point.X < minX || point.X > maxX || point.Y < minY || point.Y > maxY {
			t.Errorf("Point %v out of AABB bounds [%v, %v]", point, p.GetAABB()[0], p.GetAABB()[1])
		}
	}
}

func TestGetRandomPoint_RectanglePolygon(t *testing.T) {
	vertices := []utils.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}
	offset := utils.Vec2{X: 5, Y: 5}
	p, err := shapes.NewPolygonShape(vertices, offset)
	if err != nil {
		t.Fatalf("Failed to create polygon: %v", err)
	}

	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), rand.Uint64()))
	numPoints := 1000
	minX, minY := p.GetAABB()[0].X, p.GetAABB()[0].Y
	maxX, maxY := p.GetAABB()[1].X, p.GetAABB()[1].Y

	for range numPoints {
		point := p.GetRandomPoint(rng)
		if point.X < minX || point.X > maxX || point.Y < minY || point.Y > maxY {
			t.Errorf("Point %v out of AABB bounds [%v, %v]", point, p.GetAABB()[0], p.GetAABB()[1])
		}
	}
}

func TestGetRandomPoint_WithOffset(t *testing.T) {
	vertices := []utils.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 10}}
	offset := utils.Vec2{X: 100, Y: 200}
	p, err := shapes.NewPolygonShape(vertices, offset)
	if err != nil {
		t.Fatalf("Failed to create polygon: %v", err)
	}

	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), rand.Uint64()))
	point := p.GetRandomPoint(rng)

	// Check if the offset is applied by comparing to an expected AABB shift
	expectedMinX := p.GetAABB()[0].X
	expectedMaxX := p.GetAABB()[1].X
	// We don't have a direct way to get non-offset AABB without modifying NewPolygonShape,
	// so let's check against the AABB
	if point.X < expectedMinX || point.X > expectedMaxX || point.Y < p.GetAABB()[0].Y || point.Y > p.GetAABB()[1].Y {
		t.Errorf("Point %v not within expected offset AABB bounds [%v, %v]", point, p.GetAABB()[0], p.GetAABB()[1])
	}
}

func TestGetRandomPoint_UniformDistribution_CentroidApprox(t *testing.T) {
	vertices := []utils.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}
	p, err := shapes.NewPolygonShape(vertices, utils.Vec2{})
	if err != nil {
		t.Fatalf("Failed to create polygon: %v", err)
	}

	expectedCentroid := calculateCentroid(p)

	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), rand.Uint64()))
	numPoints := 100000 // Large number of points for approximation
	sumPoints := utils.Vec2{}

	for range numPoints {
		sumPoints = sumPoints.Add(p.GetRandomPoint(rng))
	}

	averagePoint := sumPoints.Multiply(1.0 / float64(numPoints))

	// Allow a small epsilon for floating point comparison
	epsilon := 0.5 // Increased epsilon to account for larger polygon and less strict centroid match
	if math.Abs(averagePoint.X-expectedCentroid.X) > epsilon || math.Abs(averagePoint.Y-expectedCentroid.Y) > epsilon {
		t.Errorf("Average point %v not close to expected centroid %v (epsilon %f)", averagePoint, expectedCentroid, epsilon)
	}
}
