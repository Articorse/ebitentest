package collidershapes

import (
	"ebittest/utils"
	"fmt"
)

type CircleShape struct {
	radius float64
	offset utils.Vec2
	aabb   [2]utils.Vec2
}

func (CircleShape) isShape() {}

func (x *CircleShape) GetAABB() [2]utils.Vec2 {
	return x.aabb
}

func (x *CircleShape) GetOffset() utils.Vec2 {
	return x.offset
}

func (x *CircleShape) GetRadius() float64 {
	return x.radius
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
