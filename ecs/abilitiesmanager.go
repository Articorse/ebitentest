package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
)

type AbilitiesManager struct{}

func NewAbilityDef(effect AbilityFunc, cd int, duration int, postEffect AbilityFunc) AbilityDef {
	return AbilityDef{
		Effect:     effect,
		CooldownMs: cd,
		DurationMs: duration,
		PostEffect: postEffect,
	}
}

func NewAbilitiesComponent(defs [data.MaxAbilitySlots]EntityAbility) *abilities {
	abis := [data.MaxAbilitySlots]EntityAbility{}

	for i, abi := range defs {
		abis[i] = EntityAbility{
			Name:   abi.Name,
			Def:    abi.Def,
			Status: AbilityStatus{State: AbiAct_Ready},
		}
	}

	return &abilities{
		abilities: abis,
	}
}

func (AbilitiesManager) TickAbilities(e common.EntityId, world *World) error {
	abiComp, err := world.Abilities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	for i, a := range abiComp.abilities {
		switch a.Status.State {
		case AbiAct_Active:
			a.Status.DurationCounterMs -= data.TickMs

			if a.Status.DurationCounterMs <= 0 {
				a.Status.State = AbiAct_OnCooldown

				if a.Def.PostEffect != nil {
					err := a.Def.PostEffect(e, nil, world)
					if err != nil {
						return fmt.Errorf("error executing post effect of ability %v of entity %d: %v", a.Name, e, err)
					}
				}
			}
		case AbiAct_OnCooldown:
			a.Status.CooldownCounterMs -= data.TickMs

			if a.Status.CooldownCounterMs <= 0 {
				a.Status.State = AbiAct_Ready
			}
		}

		abiComp.abilities[i] = a
	}

	return nil
}

func (AbilitiesManager) HasAbility(e common.EntityId, name AbilityEnum, world *World) bool {
	abiComp, err := world.Abilities.getComponent(e)
	if err != nil {
		return false
	}

	for _, a := range abiComp.abilities {
		if a.Name == name {
			return true
		}
	}

	return false
}

func (AbilitiesManager) DisableAbility(e common.EntityId, name AbilityEnum, world *World) error {
	abiComp, err := world.Abilities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	var abi EntityAbility
	idx := -1

	for i, a := range abiComp.abilities {
		if a.Name == name {
			abi = a
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("entity %d does not have ability %v", e, name)
	}

	abi.Status.State = AbiAct_Disabled
	abiComp.abilities[idx] = abi

	return nil
}

func (AbilitiesManager) EnableAbility(e common.EntityId, name AbilityEnum, world *World) error {
	abiComp, err := world.Abilities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	var abi EntityAbility
	idx := -1

	for i, a := range abiComp.abilities {
		if a.Name == name {
			abi = a
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("entity %d does not have ability %v", e, name)
	}

	abi.Status.State = AbiAct_Ready
	abiComp.abilities[idx] = abi

	return nil
}

func (AbilitiesManager) ActivateAbility(
	e common.EntityId,
	targets []common.EntityId,
	abiIdx int,
	world *World,
) (hasAbility bool, err error) {
	if abiIdx > data.MaxAbilitySlots-1 {
		return false, fmt.Errorf("ability index %d is out of bounds", abiIdx)
	}

	abiComp, err := world.Abilities.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	abi := abiComp.abilities[abiIdx]

	if abi.Name == Ability_None {
		return false, nil
	}

	if abi.Status.State != AbiAct_Ready {
		return false, nil
	}

	abi.Status.DurationCounterMs = abi.Def.DurationMs
	abi.Status.CooldownCounterMs = abi.Def.CooldownMs
	abi.Status.State = AbiAct_Active

	abiComp.abilities[abiIdx] = abi

	err = abi.Def.Effect(e, targets, world)
	if err != nil {
		return false, fmt.Errorf("error executing effect of ability %v of entity %d: %v", abi.Name, e, err)
	}

	return true, nil
}
