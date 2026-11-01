package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

func TestInputsAreHandledInEntityOrder(t *testing.T) {
	x := testutil.NewECS(t)
	const recordOrder = ecs.AbilityEnum(1000)
	order := []common.EntityId{}
	err := x.RegisterAbility(recordOrder, func(self common.EntityId, _ ecs.AbilityParams, _ *ecs.ECSContainer) error {
		order = append(order, self)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	inputs := map[common.EntityId]ecs.InputState{}
	for range 50 {
		abis := [data.MaxAbilitySlots]ecs.EntityAbility{{Def: ecs.NewAbilityDef(recordOrder, ecs.Ability_None, 0, 0)}}
		inputs[testutil.Add(t, x, ecs.NewAbilitiesComponent(abis))] = ecs.InputState{Ability1: 1}
	}

	if err := inputsystem.HandleInputs(utils.Vec2f{}, x, inputs); err != nil {
		t.Fatal(err)
	}

	if len(order) != len(inputs) || !slices.IsSorted(order) {
		t.Fatalf("abilities activated in order %v, want every entity in ID order", order)
	}
}
