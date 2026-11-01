package shapes_test

import (
	"math/rand/v2"
	"testing"

	"ebittest/ecs/shapes"
	"ebittest/utils"
)

func TestRectangleOffsetIsAppliedOnce(t *testing.T) {
	r, err := shapes.NewRectangleShape(10, 20, utils.Vec2f{X: 100, Y: 200})
	if err != nil {
		t.Fatal(err)
	}

	aabb := r.GetAABB()
	if want := [2]utils.Vec2f{{X: 95, Y: 190}, {X: 105, Y: 210}}; aabb != want {
		t.Fatalf("AABB %v, want %v", aabb, want)
	}

	rng := rand.New(rand.NewPCG(1, 2))
	for range 100 {
		if p := r.GetRandomPoint(rng); p.X < aabb[0].X || p.X > aabb[1].X || p.Y < aabb[0].Y || p.Y > aabb[1].Y {
			t.Fatalf("random point %v is outside the rectangle %v", p, aabb)
		}

		p := r.GetRandomPointAroundShape(rng)
		onVertical := (p.X == aabb[0].X || p.X == aabb[1].X) && p.Y >= aabb[0].Y && p.Y <= aabb[1].Y
		onHorizontal := (p.Y == aabb[0].Y || p.Y == aabb[1].Y) && p.X >= aabb[0].X && p.X <= aabb[1].X
		if !onVertical && !onHorizontal {
			t.Fatalf("point around the shape %v isn't on the rectangle's edge %v", p, aabb)
		}
	}
}

func TestRectangleSurvivesDtoRoundTrip(t *testing.T) {
	r, err := shapes.NewRectangleShape(10, 20, utils.Vec2f{X: 100, Y: 200})
	if err != nil {
		t.Fatal(err)
	}

	dto, err := shapes.ShapeToDto(r)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := shapes.DtoToShape[shapes.ShapeDto](dto)
	if err != nil {
		t.Fatal(err)
	}

	if restored.GetAABB() != r.GetAABB() || restored.GetOffset() != r.GetOffset() {
		t.Fatalf("restored AABB %v and offset %v, want %v and %v", restored.GetAABB(), restored.GetOffset(), r.GetAABB(), r.GetOffset())
	}
}
