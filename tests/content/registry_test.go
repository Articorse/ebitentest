package content_test

import (
	"ebittest/content/abilities"
	"ebittest/content/inputsources"
	"ebittest/content/timerfuncs"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"testing"
)

func noopAbility(common.EntityId, ecs.AbilityParams, *ecs.ECSContainer) error {
	return nil
}

func TestRegistryRefusesTakenAndNoneIds(t *testing.T) {
	x := testutil.NewECS(t)

	if err := x.RegisterAbility(abilities.Ability_Dodge, noopAbility); err == nil {
		t.Error("ability ID registered twice")
	}
	if err := x.RegisterAbility(ecs.Ability_None, noopAbility); err == nil {
		t.Error("ability registered as none")
	}
	if err := x.RegisterTimerFunc(timerfuncs.TimerFunc_Spawn, func(common.EntityId, *ecs.ECSContainer) error { return nil }); err == nil {
		t.Error("timer function ID registered twice")
	}
	if err := x.RegisterTimerFunc(ecs.TimerFunc_None, func(common.EntityId, *ecs.ECSContainer) error { return nil }); err == nil {
		t.Error("timer function registered as none")
	}
	if err := x.RegisterInputSource(inputsources.InputType_Human, inputsources.DummyInputSource); err == nil {
		t.Error("input source ID registered twice")
	}
}

func TestUnregisteredAbilityReportedOnUse(t *testing.T) {
	x := ecs.NewECSContainer()

	if _, err := x.AbilitiesManager.GetAbilityFunc(abilities.Ability_Dodge, x); err == nil {
		t.Fatal("unregistered ability found")
	}
	if f, err := x.AbilitiesManager.GetAbilityFunc(ecs.Ability_None, x); f != nil || err != nil {
		t.Fatalf("none ability: function %v, error %v", f != nil, err)
	}

	if err := x.RegisterAbility(abilities.Ability_Dodge, noopAbility); err != nil {
		t.Fatal(err)
	}
	if f, err := x.AbilitiesManager.GetAbilityFunc(abilities.Ability_Dodge, x); f == nil || err != nil {
		t.Fatalf("registered ability: function %v, error %v", f != nil, err)
	}
}
