package shapes

import (
	"ebittest/utils"
	"fmt"
	"math"
	"math/rand/v2"
)

type CircleShape struct {
	radius float64
	offset utils.Vec2f
	aabb   [2]utils.Vec2f
}

func NewCircleShape(r float64, o utils.Vec2f) (*CircleShape, error) {
	if r < 0 {
		return nil, fmt.Errorf("radius must be non-negative")
	}

	return &CircleShape{
		radius: r,
		offset: o,
		aabb: [2]utils.Vec2f{
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

func (x *CircleShape) GetAABB() [2]utils.Vec2f {
	return x.aabb
}

func (x *CircleShape) GetOffset() utils.Vec2f {
	return x.offset
}

func (x *CircleShape) GetRadius() float64 {
	return x.radius
}

func (x *CircleShape) GetRandomPoint(r *rand.Rand) utils.Vec2f {
	d := x.radius * r.Float64()
	a := math.Pi * 2 * r.Float64()

	return utils.Vec2f{X: d*math.Cos(a) + x.offset.X, Y: d*math.Sin(a) + x.offset.Y}
}

func (x *CircleShape) GetRandomPointAroundShape(r *rand.Rand) utils.Vec2f {
	a := math.Pi * 2 * r.Float64()

	return utils.Vec2f{X: x.radius*math.Cos(a) + x.offset.X, Y: x.radius*math.Sin(a) + x.offset.Y}
}

type CircleParams struct {
	Radius float64
	Offset utils.Vec2f
}

func (CircleParams) isShapeParams() {}
