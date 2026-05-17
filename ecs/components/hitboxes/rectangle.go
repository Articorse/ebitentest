package hitboxes

import (
	"ebittest/utils"
	"fmt"
)

type RectangleHitbox struct {
	topLeft     utils.Vec2
	offset      utils.Vec2
	bottomRight utils.Vec2
}

func (RectangleHitbox) isHitbox() {}

func (x *RectangleHitbox) GetAABB() [2]utils.Vec2 {
	return [2]utils.Vec2{x.topLeft, x.bottomRight}
}

func (x *RectangleHitbox) GetOffset() utils.Vec2 {
	return x.offset
}

func NewRectangleHitbox(w float64, h float64, o utils.Vec2) (*RectangleHitbox, error) {
	if w < 0 || h < 0 {
		return nil, fmt.Errorf("width and height must be non-negative")
	}

	return &RectangleHitbox{
		topLeft:     utils.Vec2{X: -w/2 + o.X, Y: -h/2 + o.Y},
		bottomRight: utils.Vec2{X: w/2 + o.X, Y: h/2 + o.Y},
		offset:      o,
	}, nil
}
