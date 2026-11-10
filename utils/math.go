package utils

import (
	"fmt"
	"math"
)

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

// Whole milligrams as the largest unit they make at least a tenth of, to a tenth, rounded down and without a trailing
// .0: 1100000 is 1.1kg, 100000 is 0.1kg, 1100 is 1.1g, 100 is 0.1g and 10 is 10mg
func FormatMass(mg int) string {
	tenths := func(n, unit int, name string) string {
		t := n * 10 / unit
		if t%10 == 0 {
			return fmt.Sprintf("%d%s", t/10, name)
		}
		return fmt.Sprintf("%d.%d%s", t/10, t%10, name)
	}
	switch {
	case mg >= 100_000:
		return tenths(mg, 1_000_000, "kg")
	case mg >= 100:
		return tenths(mg, 1_000, "g")
	}
	return fmt.Sprintf("%dmg", mg)
}
