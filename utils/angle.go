package utils

import "math"

// The same angle in [-π, π]
func NormalizeAngle(a float64) float64 {
	return math.Remainder(a, 2*math.Pi)
}
