package shapes

import (
	"ebittest/utils"
	"fmt"
	"math"
	"math/rand/v2"
)

type CircleShape struct {
	radius float64
	offset utils.Vec2
	aabb   [2]utils.Vec2
}

func NewCircleShape(r float64, o utils.Vec2) (*CircleShape, error) {
	if r < 0 {
		return nil, fmt.Errorf("radius must be non-negative")
	}

	return &CircleShape{
		radius: r,
		offset: o,
		aabb: [2]utils.Vec2{
			{X: -r + o.X, Y: -r + o.Y},
			{X: r + o.X, Y: r + o.Y},
		},
	}, nil
}

func (x *CircleShape) Copy() Shape {
	return &CircleShape{
		radius: x.radius,
		offset: x.offset,
		aabb:   x.aabb,
	}
}

func (x *CircleShape) GetAABB() [2]utils.Vec2 {
	return x.aabb
}

func (x *CircleShape) GetOffset() utils.Vec2 {
	return x.offset
}

func (x *CircleShape) GetRadius() float64 {
	return x.radius
}

func (x *CircleShape) GetRandomPoint(r rand.Rand) utils.Vec2 {
	d := x.radius * r.Float64()
	a := math.Pi * 2 * r.Float64()

	return utils.Vec2{X: d * math.Cos(a), Y: d * math.Sin(a)}
}

func (x *CircleShape) GetRandomPointAroundShape(r rand.Rand) utils.Vec2 {
	a := math.Pi * 2 * r.Float64()

	return utils.Vec2{X: x.radius * math.Cos(a), Y: x.radius * math.Sin(a)}
}
