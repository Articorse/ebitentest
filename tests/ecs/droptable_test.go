package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"math"
	"testing"
)

func wood(n int) []ecs.ItemDrop {
	return []ecs.ItemDrop{{Kind: data.Item_Wood, Count: n}}
}

func TestDropTablesRefuseOutcomesThatDontAddUp(t *testing.T) {
	for _, c := range []struct {
		name     string
		outcomes []ecs.DropOutcome
		ok       bool
	}{
		{"exactly 100%", []ecs.DropOutcome{{ChancePercent: 60, Items: wood(1)}, {ChancePercent: 40, Items: wood(2)}}, true},
		{"under 100%", []ecs.DropOutcome{{ChancePercent: 30, Items: wood(2)}, {ChancePercent: 50, Items: wood(1)}}, true},
		{"two outcomes with the same chance", []ecs.DropOutcome{{ChancePercent: 30, Items: wood(2)}, {ChancePercent: 30, Items: wood(1)}}, true},
		{"over 100%", []ecs.DropOutcome{{ChancePercent: 60, Items: wood(1)}, {ChancePercent: 40.5, Items: wood(2)}}, false},
		{"a negative chance", []ecs.DropOutcome{{ChancePercent: -10, Items: wood(1)}, {ChancePercent: 50, Items: wood(2)}}, false},
		{"dropping none of an item", []ecs.DropOutcome{{ChancePercent: 50, Items: wood(0)}}, false},
	} {
		if _, err := ecs.NewDropTable(c.outcomes...); (err == nil) != c.ok {
			t.Errorf("%s: error %v, want ok %v", c.name, err, c.ok)
		}
	}
}

// How many wood a roll of the table gives
func rollWood(x *ecs.ECSContainer, table *ecs.DropTable) int {
	n := 0
	for _, d := range table.Roll(x.Rng) {
		n += d.Count
	}
	return n
}

func TestDropTablesRollEachOutcomeAsOftenAsItsChance(t *testing.T) {
	x := testutil.NewECS(t)
	table, err := ecs.NewDropTable(ecs.DropOutcome{ChancePercent: 30, Items: wood(2)}, ecs.DropOutcome{ChancePercent: 50, Items: wood(1)})
	if err != nil {
		t.Fatal(err)
	}

	const rolls = 4000
	counts := map[int]int{}
	for range rolls {
		counts[rollWood(x, table)]++
	}

	for dropped, want := range map[int]float64{2: 0.3, 1: 0.5, 0: 0.2} {
		if got := float64(counts[dropped]) / rolls; math.Abs(got-want) > 0.03 {
			t.Errorf("dropped %d in %.1f%% of rolls, want about %.0f%%", dropped, got*100, want*100)
		}
	}
	if len(counts) != 3 {
		t.Errorf("drop counts %v, want only 0, 1 and 2", counts)
	}
}
