package utils_test

import (
	"ebittest/utils"
	"testing"
)

func TestAMassIsShownInTheLargestUnitItMakesATenthOfToATenthRoundedDown(t *testing.T) {
	for _, c := range []struct {
		mg   int
		want string
	}{
		{1000000, "1kg"}, {1001000, "1kg"}, {1010000, "1kg"}, {1100000, "1.1kg"}, {100000, "0.1kg"},
		{1000, "1g"}, {1001, "1g"}, {1010, "1g"}, {1100, "1.1g"}, {100, "0.1g"},
		{10, "10mg"}, {1, "1mg"}, {0, "0mg"}, {99999, "99.9g"}, {2500000000, "2500kg"},
	} {
		if got := utils.FormatMass(c.mg); got != c.want {
			t.Errorf("%d mg is shown as %q, want %q", c.mg, got, c.want)
		}
	}
}
