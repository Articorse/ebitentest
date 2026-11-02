package shapes_test

import (
	"math"
	"testing"

	"ebittest/ecs/shapes"
	"ebittest/utils"
)

func near(a, b utils.Vec2f) bool {
	return math.Abs(a.X-b.X) < 1e-9 && math.Abs(a.Y-b.Y) < 1e-9
}

func TestAnUnturnedUnscaledShapeIsLeftAsItIs(t *testing.T) {
	c, _ := shapes.NewCircleShape(3, utils.Vec2f{X: 1})
	if shapes.Transform(c, 0, 1) != shapes.Shape(c) {
		t.Fatal("a shape that isn't turned or scaled was copied")
	}
}

func TestCirclesTurnTheirOffsetAndScaleTheirRadius(t *testing.T) {
	c, _ := shapes.NewCircleShape(3, utils.Vec2f{X: 10})
	got := shapes.Transform(c, math.Pi/2, 2).(*shapes.CircleShape)

	if !near(got.GetOffset(), utils.Vec2f{Y: 20}) || got.GetRadius() != 6 {
		t.Fatalf("turned a quarter clockwise and doubled, the circle is at %v with radius %v, want at {0 20} with 6",
			got.GetOffset(), got.GetRadius())
	}
	if aabb := got.GetAABB(); !near(aabb[0], utils.Vec2f{X: -6, Y: 14}) || !near(aabb[1], utils.Vec2f{X: 6, Y: 26}) {
		t.Fatalf("the turned circle's AABB is %v", aabb)
	}
}

func TestRectanglesStayAxisAlignedAndOnlyTurnTheirOffset(t *testing.T) {
	r, _ := shapes.NewRectangleShape(10, 4, utils.Vec2f{X: 10})
	got := shapes.Transform(r, math.Pi/4, 2).(*shapes.RectangleShape)

	offset := utils.Vec2f{X: 20 / math.Sqrt2, Y: 20 / math.Sqrt2}
	if !near(got.GetOffset(), offset) {
		t.Fatalf("the rectangle's offset is %v, want %v", got.GetOffset(), offset)
	}
	aabb := got.GetAABB()
	if size := aabb[1].Subtract(aabb[0]); !near(size, utils.Vec2f{X: 20, Y: 8}) {
		t.Fatalf("the turned rectangle is %v, want 20×8: scaled, but not turned", size)
	}
}

func TestPolygonsTurnAndScaleTheirVerticesAndOffset(t *testing.T) {
	p, _ := shapes.NewPolygonShape([]utils.Vec2f{{X: 0}, {X: 4}, {X: 4, Y: 2}}, utils.Vec2f{X: 1})
	got := shapes.Transform(p, math.Pi/2, 3).(*shapes.PolygonShape)

	want := []utils.Vec2f{{X: 0}, {Y: 12}, {X: -6, Y: 12}}
	for i, v := range got.GetVertices() {
		if !near(v, want[i]) {
			t.Fatalf("the vertices are %v, want %v", got.GetVertices(), want)
		}
	}
	if !near(got.GetOffset(), utils.Vec2f{Y: 3}) {
		t.Fatalf("the offset is %v, want {0 3}", got.GetOffset())
	}
	if aabb := got.GetAABB(); !near(aabb[0], utils.Vec2f{X: -6, Y: 3}) || !near(aabb[1], utils.Vec2f{Y: 15}) {
		t.Fatalf("the AABB is %v", aabb)
	}
	if v := p.GetVertices()[1]; v != (utils.Vec2f{X: 4}) {
		t.Fatal("transforming the polygon changed the original")
	}
}
