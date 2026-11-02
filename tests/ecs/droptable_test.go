package ecs_test

import (
	"ebittest/content/items"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/systems/itemsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

func planks(n int) []ecs.ItemDrop {
	return []ecs.ItemDrop{{Kind: items.ItemKind_Planks, Count: n}}
}

func TestDropTablesRefuseOutcomesThatDontAddUp(t *testing.T) {
	for _, c := range []struct {
		name     string
		outcomes []ecs.DropOutcome
		ok       bool
	}{
		{"exactly 100%", []ecs.DropOutcome{{ChancePercent: 60, Items: planks(1)}, {ChancePercent: 40, Items: planks(2)}}, true},
		{"under 100%", []ecs.DropOutcome{{ChancePercent: 30, Items: planks(2)}, {ChancePercent: 50, Items: planks(1)}}, true},
		{"two outcomes with the same chance", []ecs.DropOutcome{{ChancePercent: 30, Items: planks(2)}, {ChancePercent: 30, Items: planks(1)}}, true},
		{"over 100%", []ecs.DropOutcome{{ChancePercent: 60, Items: planks(1)}, {ChancePercent: 40.5, Items: planks(2)}}, false},
		{"a negative chance", []ecs.DropOutcome{{ChancePercent: -10, Items: planks(1)}, {ChancePercent: 50, Items: planks(2)}}, false},
		{"dropping none of an item", []ecs.DropOutcome{{ChancePercent: 50, Items: planks(0)}}, false},
	} {
		if _, err := ecs.NewDropTable(c.outcomes...); (err == nil) != c.ok {
			t.Errorf("%s: error %v, want ok %v", c.name, err, c.ok)
		}
	}
}

// Kills an entity with the table at the origin and returns how many planks it dropped
func killAndCountDrops(t *testing.T, x *ecs.ECSContainer, table *ecs.DropTable) int {
	t.Helper()
	before := len(x.Items.GetEntities())
	e := testutil.AddMovable(t, x, utils.Vec2f{}, false, table)
	x.ScheduleKillEntity(e)
	testutil.RemoveScheduled(t, x)
	return len(x.Items.GetEntities()) - before
}

func TestDropTablesRollEachOutcomeAsOftenAsItsChance(t *testing.T) {
	x := testutil.NewECS(t)
	table, err := ecs.NewDropTable(ecs.DropOutcome{ChancePercent: 30, Items: planks(2)}, ecs.DropOutcome{ChancePercent: 50, Items: planks(1)})
	if err != nil {
		t.Fatal(err)
	}

	const rolls = 4000
	counts := map[int]int{}
	for range rolls {
		counts[killAndCountDrops(t, x, table)]++
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

func TestOnlyKilledEntitiesDropTheirLoot(t *testing.T) {
	x := testutil.NewECS(t)
	table, err := ecs.NewDropTable(ecs.DropOutcome{ChancePercent: 100, Items: planks(1)})
	if err != nil {
		t.Fatal(err)
	}
	pos := utils.Vec2f{X: 30, Y: -20}

	removed := testutil.AddMovable(t, x, pos, false, table)
	x.ScheduleRemoveEntity(removed)
	testutil.RemoveScheduled(t, x)
	if n := len(x.Items.GetEntities()); n != 0 {
		t.Fatalf("an entity removed without being killed dropped %d items", n)
	}

	killed := testutil.AddMovable(t, x, pos, false, table)
	x.ScheduleKillEntity(killed)
	testutil.RemoveScheduled(t, x)
	dropped := x.Items.GetEntities()
	if len(dropped) != 1 || testutil.WorldPos(t, x, dropped[0]) != pos || !x.Tosses.HasComponent(dropped[0]) {
		t.Fatalf("a killed entity dropped %v, want one plank tossed from where it died, %v", dropped, pos)
	}
}

func TestDropTablesAreSaved(t *testing.T) {
	x := testutil.NewECS(t)
	table, err := ecs.NewDropTable(ecs.DropOutcome{ChancePercent: 100, Items: planks(3)})
	if err != nil {
		t.Fatal(err)
	}
	e := testutil.AddMovable(t, x, utils.Vec2f{}, false, table)

	members, err := x.GetCompositionComponents(e)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)
	y.ScheduleKillEntity(e)
	testutil.RemoveScheduled(t, y)
	if n := len(y.Items.GetEntities()); n != 3 {
		t.Fatalf("after loading, the entity dropped %d planks, want 3", n)
	}
}

func TestEachDropIsTossedItsOwnWay(t *testing.T) {
	x := testutil.NewECS(t)
	table, err := ecs.NewDropTable(ecs.DropOutcome{ChancePercent: 100, Items: planks(3)})
	if err != nil {
		t.Fatal(err)
	}
	e := testutil.AddMovable(t, x, utils.Vec2f{}, false, table)
	x.ScheduleKillEntity(e)
	testutil.RemoveScheduled(t, x)

	for range data.ItemTossDurationMs/data.TickMs + 1 {
		itemsystem.Tick(x)
	}

	angles := []float64{}
	for _, item := range x.Items.GetEntities() {
		pos := testutil.WorldPos(t, x, item)
		angles = append(angles, math.Atan2(pos.Y, pos.X))
	}
	if len(angles) != 3 || angles[0] == angles[1] || angles[1] == angles[2] || angles[0] == angles[2] {
		t.Fatalf("the drops landed in directions %v, want three different ones", angles)
	}
}
