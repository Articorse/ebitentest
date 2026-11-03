package utils

import "math"

func AbsInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func Clamp(a, min, max float64) float64 {
	if a < min {
		return min
	}
	if a > max {
		return max
	}
	return a
}

func Sign(a float64) float64 {
	if a > 0 {
		return 1
	}
	if a < 0 {
		return -1
	}
	return 0
}

func Radian2Degree(a float64) float64 {
	return a * 180 / math.Pi
}

func Degree2Radian(a float64) float64 {
	return a * math.Pi / 180
}

// The smallest integer whose square is at least a, worked out exactly for any non-negative a
func CeilSqrt(a int) int {
	r := int(math.Sqrt(float64(a)))
	for r*r > a {
		r--
	}
	for r*r < a {
		r++
	}
	return r
}
