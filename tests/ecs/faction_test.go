package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestColonyAndMonstersAreHostileEitherWayRound(t *testing.T) {
	if !ecs.AreHostile(data.Faction_Colony, data.Faction_Monsters) || !ecs.AreHostile(data.Faction_Monsters, data.Faction_Colony) {
		t.Fatal("the colony and monsters aren't hostile to each other")
	}
	for _, pair := range [][2]data.FactionEnum{
		{data.Faction_Colony, data.Faction_Colony}, {data.Faction_Monsters, data.Faction_Monsters}, {data.Faction_None, data.Faction_Monsters},
	} {
		if ecs.AreHostile(pair[0], pair[1]) {
			t.Errorf("factions %v and %v are hostile", pair[0], pair[1])
		}
	}
}

// A monster at the origin that sees everything within 20 tiles
func watchingMonster(t *testing.T, x *ecs.ECSContainer) common.EntityId {
	t.Helper()
	m := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewFactionComponent(data.Faction_Monsters), ecs.NewVisionComponent(20, 1))
	v, _ := x.Visions.Get(m)
	v.Update(utils.Vec2i{}, 20, func(utils.Vec2i) bool { return false }, func(utils.Vec2i) uint8 { return data.MaxLight })
	return m
}

func addFighter(t *testing.T, x *ecs.ECSContainer, tile utils.Vec2i, faction data.FactionEnum) common.EntityId {
	t.Helper()
	return testutil.AddMovable(t, x, utils.TilePosToWorldPos(tile), false, ecs.NewFactionComponent(faction), ecs.NewBodyComponent(data.Body_Humanoid))
}

func TestTheNearestVisibleHostileIsChosen(t *testing.T) {
	x := testutil.NewECS(t)
	monster := watchingMonster(t, x)
	if _, ok := x.NearestVisibleHostile(monster); ok {
		t.Fatal("a target was found with nobody around")
	}

	far := addFighter(t, x, utils.Vec2i{X: 10}, data.Faction_Colony)
	addFighter(t, x, utils.Vec2i{X: 2}, data.Faction_Monsters)
	addFighter(t, x, utils.Vec2i{X: -1}, data.Faction_None)
	addFighter(t, x, utils.Vec2i{Y: 25}, data.Faction_Colony)
	testutil.AddMovable(t, x, utils.TilePosToWorldPos(utils.Vec2i{X: 1}), false, ecs.NewFactionComponent(data.Faction_Colony))
	if got, ok := x.NearestVisibleHostile(monster); !ok || got != far {
		t.Fatalf("the target is %d (%v), want the only seen colonist with a body, %d", got, ok, far)
	}

	near := addFighter(t, x, utils.Vec2i{Y: -4}, data.Faction_Colony)
	tied := addFighter(t, x, utils.Vec2i{X: 4}, data.Faction_Colony)
	if got, _ := x.NearestVisibleHostile(monster); got != near || near > tied {
		t.Fatalf("the target is %d, want the closest, %d, which also wins the tie with %d", got, near, tied)
	}

	x.SetUntargetable(near, true)
	x.SetDisabled(tied, true)
	if got, _ := x.NearestVisibleHostile(monster); got != far {
		t.Fatalf("with the closest untargetable and the next disabled, the target is %d, want %d", got, far)
	}
	x.SetUntargetable(near, false)
	if got, _ := x.NearestVisibleHostile(monster); got != near {
		t.Fatalf("once targetable again, the closest isn't the target: got %d, want %d", got, near)
	}
}

func TestTheFactionIsSaved(t *testing.T) {
	x := testutil.NewECS(t)
	e := addFighter(t, x, utils.Vec2i{}, data.Faction_Monsters)
	members, err := x.GetCompositionComponents(e)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)
	if f, ok := y.Factions.Get(e); !ok || f.Faction() != data.Faction_Monsters {
		t.Fatal("the restored entity isn't a monster")
	}
}
