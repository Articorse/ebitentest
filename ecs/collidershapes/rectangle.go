package collidershapes

import (
	"ebittest/utils"
	"fmt"
)

type RectangleShape struct {
	topLeft     utils.Vec2
	bottomRight utils.Vec2
	offset      utils.Vec2
}

func (RectangleShape) isShape() {}

func (x *RectangleShape) GetAABB() [2]utils.Vec2 {
	return [2]utils.Vec2{x.topLeft, x.bottomRight}
}

func (x *RectangleShape) GetOffset() utils.Vec2 {
	return x.offset
}

func NewRectangleShape(w float64, h float64, o utils.Vec2) (*RectangleShape, error) {
	if w < 0 || h < 0 {
		return nil, fmt.Errorf("width and height must be non-negative")
	}

	return &RectangleShape{
		topLeft:     utils.Vec2{X: -w/2 + o.X, Y: -h/2 + o.Y},
		bottomRight: utils.Vec2{X: w/2 + o.X, Y: h/2 + o.Y},
		offset:      o,
	}, nil
}
