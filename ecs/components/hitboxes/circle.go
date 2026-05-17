package hitboxes

import (
	"ebittest/utils"
	"fmt"
)

type CircleHitbox struct {
	radius float64
	offset utils.Vec2
	aabb   [2]utils.Vec2
}

func (CircleHitbox) isHitbox() {}

func (x *CircleHitbox) GetAABB() [2]utils.Vec2 {
	return x.aabb
}

func (x *CircleHitbox) GetOffset() utils.Vec2 {
	return x.offset
}

func (x *CircleHitbox) GetRadius() float64 {
	return x.radius
}

func NewCircleHitbox(r float64, o utils.Vec2) (*CircleHitbox, error) {
	if r < 0 {
		return nil, fmt.Errorf("radius must be non-negative")
	}

	return &CircleHitbox{
		radius: r,
		offset: o,
		aabb: [2]utils.Vec2{
			{X: -r + o.X, Y: -r + o.Y},
			{X: r + o.X, Y: r + o.Y},
		},
	}, nil
}
