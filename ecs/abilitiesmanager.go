package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type abilitiesManager struct{}

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

func (abilitiesManager) TickAbilities(e common.EntityId, ecsContainer *ECSContainer) error {
	abiComp, err := ecsContainer.Abilities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	for i, a := range abiComp.abilities {
		err := tickAbilityState(e, &a, ecsContainer)
		if err != nil {
			return fmt.Errorf("error ticking ability %v of entity %d: %v", a.Name, e, err)
		}

		abiComp.abilities[i] = a
	}

	return nil
}

func (abilitiesManager) HasAbility(e common.EntityId, name AbilityEnum, ecsContainer *ECSContainer) bool {
	abiComp, err := ecsContainer.Abilities.getComponent(e)
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

func (abilitiesManager) DisableAbility(e common.EntityId, name AbilityEnum, ecsContainer *ECSContainer) error {
	abiComp, err := ecsContainer.Abilities.getComponent(e)
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

func (abilitiesManager) EnableAbility(e common.EntityId, name AbilityEnum, ecsContainer *ECSContainer) error {
	abiComp, err := ecsContainer.Abilities.getComponent(e)
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

func (abilitiesManager) ActivateAbility(
	e common.EntityId,
	targets []common.EntityId,
	targetPos utils.Vec2,
	abiIdx int,
	ecsContainer *ECSContainer,
) (activated bool, err error) {
	if abiIdx > data.MaxAbilitySlots-1 {
		return false, fmt.Errorf("ability index %d is out of bounds", abiIdx)
	}

	abiComp, err := ecsContainer.Abilities.getComponent(e)
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

	_, err = tryActivate(e, &abi, targets, targetPos, ecsContainer)
	if err != nil {
		return false, fmt.Errorf("error trying to activate ability %v of entity %d: %v", abi.Name, e, err)
	}

	abiComp.abilities[abiIdx] = abi

	return true, nil
}
