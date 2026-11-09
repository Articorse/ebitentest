package shapes_test

import (
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"testing"
)

func TestAConcavePolygonIsRefused(t *testing.T) {
	arrow := []utils.Vec2f{{X: 0, Y: 0}, {X: 10, Y: 5}, {X: 0, Y: 10}, {X: 3, Y: 5}}
	if p, err := shapes.NewPolygonShape(arrow, utils.Vec2f{}); err == nil || p != nil {
		t.Fatal("a concave polygon was made")
	}
	square := []utils.Vec2f{{X: 0, Y: 0}, {X: 5, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}
	if _, err := shapes.NewPolygonShape(square, utils.Vec2f{}); err != nil {
		t.Fatalf("a square with a corner on one side was refused: %v", err)
	}
}

func TestAShapeOverlapsABoxOnlyWhereItReachesInto(t *testing.T) {
	circle, _ := shapes.NewCircleShape(5, utils.Vec2f{X: 10, Y: 10})
	diamond, err := shapes.NewPolygonShape([]utils.Vec2f{{X: 0, Y: -5}, {X: 5, Y: 0}, {X: 0, Y: 5}, {X: -5, Y: 0}}, utils.Vec2f{X: 10, Y: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name   string
		shape  shapes.Shape
		lo, hi utils.Vec2f
		want   bool
	}{
		{"a circle in its box's corner", circle, utils.Vec2f{X: 14, Y: 14}, utils.Vec2f{X: 20, Y: 20}, false},
		{"a circle over the box's edge", circle, utils.Vec2f{X: 12, Y: 0}, utils.Vec2f{X: 20, Y: 20}, true},
		{"a circle touching the box", circle, utils.Vec2f{X: 15, Y: 0}, utils.Vec2f{X: 20, Y: 20}, false},
		{"a diamond in its box's corner", diamond, utils.Vec2f{X: 13, Y: 13}, utils.Vec2f{X: 20, Y: 20}, false},
		{"a diamond's point in the box", diamond, utils.Vec2f{X: 14, Y: 8}, utils.Vec2f{X: 20, Y: 12}, true},
		{"a diamond's point touching the box", diamond, utils.Vec2f{X: 15, Y: 8}, utils.Vec2f{X: 20, Y: 12}, false},
	} {
		if got := shapes.OverlapsBox(c.shape, c.lo, c.hi); got != c.want {
			t.Errorf("%s: overlaps %v, want %v", c.name, got, c.want)
		}
	}
}
