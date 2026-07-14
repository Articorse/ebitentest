package shapes

import (
	"ebittest/utils"
	"fmt"
	"math/rand/v2"
)

type RectangleShape struct {
	topLeft     utils.Vec2f
	bottomRight utils.Vec2f
	offset      utils.Vec2f
}

func (x *RectangleShape) Copy() Shape {
	return &RectangleShape{
		topLeft:     x.topLeft,
		bottomRight: x.bottomRight,
		offset:      x.offset,
	}
}

func NewRectangleShape(w float64, h float64, o utils.Vec2f) (*RectangleShape, error) {
	if w < 0 || h < 0 {
		return nil, fmt.Errorf("width and height must be non-negative")
	}

	return &RectangleShape{
		topLeft:     utils.Vec2f{X: -w/2 + o.X, Y: -h/2 + o.Y},
		bottomRight: utils.Vec2f{X: w/2 + o.X, Y: h/2 + o.Y},
		offset:      o,
	}, nil
}

func NewRectangleShapeFromCoords(topLeft utils.Vec2f, bottomRight utils.Vec2f, o utils.Vec2f) (*RectangleShape, error) {
	if bottomRight.X < topLeft.X || bottomRight.Y < topLeft.Y {
		return nil, fmt.Errorf("bottomRight must be greater than or equal to topLeft")
	}

	return &RectangleShape{
		topLeft:     utils.Vec2f{X: topLeft.X, Y: topLeft.Y},
		bottomRight: utils.Vec2f{X: bottomRight.X, Y: bottomRight.Y},
		offset:      o,
	}, nil
}

func (x *RectangleShape) GetAABB() [2]utils.Vec2f {
	return [2]utils.Vec2f{x.topLeft, x.bottomRight}
}

func (x *RectangleShape) GetOffset() utils.Vec2f {
	return x.offset
}

func (x *RectangleShape) GetRandomPoint(r *rand.Rand) utils.Vec2f {
	xDiff := x.bottomRight.X - x.topLeft.X
	yDiff := x.bottomRight.Y - x.topLeft.Y

	xRand := r.Float64() * xDiff
	yRand := r.Float64() * yDiff

	return utils.Vec2f{X: x.topLeft.X + xRand + x.offset.X, Y: x.topLeft.Y + yRand + x.offset.Y}
}

func (x *RectangleShape) GetRandomPointAroundShape(r *rand.Rand) utils.Vec2f {
	sideLengths := []float64{
		x.bottomRight.X - x.topLeft.X,
		x.bottomRight.Y - x.topLeft.Y,
		x.bottomRight.X - x.topLeft.X,
		x.bottomRight.Y - x.topLeft.Y,
	}

	totalLength := 0.0
	for _, s := range sideLengths {
		totalLength += s
	}

	dirs := []utils.Vec2f{
		{X: 1, Y: 0},
		{X: 0, Y: 1},
		{X: -1, Y: 0},
		{X: 0, Y: -1},
	}

	sideCenters := []utils.Vec2f{
		{X: 0, Y: 0},
		{X: sideLengths[0], Y: 0},
		{X: sideLengths[0], Y: sideLengths[1]},
		{X: 0, Y: sideLengths[1]},
	}

	randLength := r.Float64() * totalLength

	for i, s := range sideLengths {
		if randLength > s {
			randLength -= s
			continue
		}
		return x.topLeft.Add(x.offset.Add(sideCenters[i].Add(dirs[i].Multiply(randLength))))
	}

	return utils.Vec2f{X: 0, Y: 0}
}

type RectangleParams struct {
	TopLeft     utils.Vec2f
	BottomRight utils.Vec2f
	Offset      utils.Vec2f
}

func (RectangleParams) isShapeParams() {}
