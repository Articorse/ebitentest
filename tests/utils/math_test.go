package utils_test

import (
	"ebittest/utils"
	"testing"
)

func TestCeilSqrt(t *testing.T) {
	for a, want := range map[int]int{0: 0, 1: 1, 2: 2, 4: 2, 5: 3, 99: 10, 100: 10, 101: 11, 22500 * 200: 2122} {
		if got := utils.CeilSqrt(a); got != want {
			t.Errorf("⌈√%d⌉ = %d, want %d", a, got, want)
		}
	}
	for a := range 10000 {
		r := utils.CeilSqrt(a)
		if r*r < a || (r > 0 && (r-1)*(r-1) >= a) {
			t.Fatalf("⌈√%d⌉ = %d", a, r)
		}
	}
}
