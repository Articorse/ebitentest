package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"math"
	"testing"
)

func wood(n int) []ecs.ItemDrop {
	return []ecs.ItemDrop{{Kind: data.Item_Wood, Amount: n}}
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
		{"dropping none to some", []ecs.DropOutcome{{ChancePercent: 50, Items: []ecs.ItemDrop{{Kind: data.Item_Wood, Max: 2}}}}, true},
		{"dropping a negative amount", []ecs.DropOutcome{{ChancePercent: 50, Items: []ecs.ItemDrop{{Kind: data.Item_Wood, Amount: -1, Max: 2}}}}, false},
		{"a range ending below where it starts", []ecs.DropOutcome{{ChancePercent: 50, Items: []ecs.ItemDrop{{Kind: data.Item_Wood, Amount: 3, Max: 2}}}}, false},
	} {
		if _, err := ecs.NewDropTable(c.outcomes...); (err == nil) != c.ok {
			t.Errorf("%s: error %v, want ok %v", c.name, err, c.ok)
		}
	}
}

// How much wood a roll of the table gives
func rollWood(x *ecs.ECSContainer, table *ecs.DropTable) int {
	n := 0
	for _, d := range table.Roll(x.Rng) {
		n += d.Amount
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

func TestADropRangeIsRolledEvenlyAndDropsNothingWhenItComesToNone(t *testing.T) {
	x := testutil.NewECS(t)
	table, err := ecs.NewDropTable(ecs.DropOutcome{ChancePercent: 100, Items: []ecs.ItemDrop{{Kind: data.Item_Wood, Max: 3}}})
	if err != nil {
		t.Fatal(err)
	}

	const rolls = 4000
	counts := map[int]int{}
	for range rolls {
		drops := table.Roll(x.Rng)
		if len(drops) > 1 || len(drops) == 1 && drops[0].Amount == 0 {
			t.Fatalf("a roll dropped %v, want one drop of some wood or none", drops)
		}
		if len(drops) == 0 {
			counts[0]++
		} else {
			counts[drops[0].Amount]++
		}
	}
	for dropped := range 4 {
		if got := float64(counts[dropped]) / rolls; math.Abs(got-0.25) > 0.03 {
			t.Errorf("dropped %d in %.1f%% of rolls, want about 25%%", dropped, got*100)
		}
	}
	if len(counts) != 4 {
		t.Errorf("drop counts %v, want only 0 to 3", counts)
	}
}
