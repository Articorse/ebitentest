package utils

import (
	"math"
)

type Vec2f struct {
	X float64
	Y float64
}

type Vec2i struct {
	X int
	Y int
}

func (v Vec2f) Normalized() Vec2f {
	l := math.Sqrt(v.X*v.X + v.Y*v.Y)

	if l <= 0 {
		return Vec2f{X: 0, Y: 0}
	}

	v.X /= l
	v.Y /= l

	return v
}

func (a Vec2f) Add(b Vec2f) Vec2f {
	return Vec2f{
		X: a.X + b.X,
		Y: a.Y + b.Y,
	}
}

func (a Vec2f) Subtract(b Vec2f) Vec2f {
	return Vec2f{
		X: a.X - b.X,
		Y: a.Y - b.Y,
	}
}

func (v Vec2f) Multiply(f float64) Vec2f {
	return Vec2f{
		X: v.X * f,
		Y: v.Y * f,
	}
}

func (v Vec2f) Dot(v2 Vec2f) float64 {
	return v.X*v2.X + v.Y*v2.Y
}

func (v Vec2f) SameDirection(v2 Vec2f) bool {
	return v.X*v2.X >= 0 && v.Y*v2.Y >= 0
}

func (v Vec2f) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func (v Vec2f) IsZero() bool {
	return v.X == 0 && v.Y == 0
}

func SegmentsIntersect(a1, a2, b1, b2 Vec2f) bool {
	return CCW(a1, b1, b2) != CCW(a2, b1, b2) && CCW(a1, a2, b1) != CCW(a1, a2, b2)
}

func CCW(a, b, c Vec2f) bool {
	return (c.Y-a.Y)*(b.X-a.X) > (b.Y-a.Y)*(c.X-a.X)
}
