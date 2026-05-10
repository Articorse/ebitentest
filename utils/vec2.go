package utils

import (
	"math"
)

type Vec2 struct {
	X float64
	Y float64
}

func (v Vec2) Normalized() Vec2 {
	l := math.Sqrt(v.X*v.X + v.Y*v.Y)

	if l <= 0 {
		return Vec2{X: 0, Y: 0}
	}

	v.X /= l
	v.Y /= l

	return v
}

func (a Vec2) Add(b Vec2) Vec2 {
	return Vec2{
		X: a.X + b.X,
		Y: a.Y + b.Y,
	}
}

func (a Vec2) Subtract(b Vec2) Vec2 {
	return Vec2{
		X: a.X - b.X,
		Y: a.Y - b.Y,
	}
}

func (v Vec2) Multiply(f float64) Vec2 {
	return Vec2{
		X: v.X * f,
		Y: v.Y * f,
	}
}

func SegmentsIntersect(a1, a2, b1, b2 Vec2) bool {
	return CCW(a1, b1, b2) != CCW(a2, b1, b2) && CCW(a1, a2, b1) != CCW(a1, a2, b2)
}

func CCW(a, b, c Vec2) bool {
	return (c.Y-a.Y)*(b.X-a.X) > (b.Y-a.Y)*(c.X-a.X)
}
