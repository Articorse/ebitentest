package utils

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
