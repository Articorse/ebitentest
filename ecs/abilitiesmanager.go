package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type abilitiesManager struct{}

func NewAbilityDef(id AbilityEnum, postId AbilityEnum, cooldownMs int, durationMs int) AbilityDef {
	return AbilityDef{
		AbilityId:     id,
		PostAbilityId: postId,
		CooldownMs:    cooldownMs,
		DurationMs:    durationMs,
	}
}

func NewAbilitiesComponent(defs [data.MaxAbilitySlots]EntityAbility) *abilities {
	abis := [data.MaxAbilitySlots]EntityAbility{}

	for i, abi := range defs {
		abis[i] = EntityAbility{
			Def:    abi.Def,
			Status: AbilityStatus{State: AbiAct_Ready},
			Params: abi.Params,
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
			return fmt.Errorf("error ticking ability %v of entity %d: %v", a.Def.AbilityId, e, err)
		}

		abiComp.abilities[i] = a
	}

	return nil
}

func (abilitiesManager) HasAbility(e common.EntityId, id AbilityEnum, ecsContainer *ECSContainer) bool {
	abiComp, err := ecsContainer.Abilities.getComponent(e)
	if err != nil {
		return false
	}

	for _, a := range abiComp.abilities {
		if a.Def.AbilityId == id {
			return true
		}
	}

	return false
}

func (abilitiesManager) DisableAbility(e common.EntityId, id AbilityEnum, ecsContainer *ECSContainer) error {
	abiComp, err := ecsContainer.Abilities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	var abi EntityAbility
	idx := -1

	for i, a := range abiComp.abilities {
		if a.Def.AbilityId == id {
			abi = a
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("entity %d does not have ability %v", e, id)
	}

	abi.Status.State = AbiAct_Disabled
	abiComp.abilities[idx] = abi

	return nil
}

func (abilitiesManager) EnableAbility(e common.EntityId, id AbilityEnum, ecsContainer *ECSContainer) error {
	abiComp, err := ecsContainer.Abilities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	var abi EntityAbility
	idx := -1

	for i, a := range abiComp.abilities {
		if a.Def.AbilityId == id {
			abi = a
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("entity %d does not have ability %v", e, id)
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

	if abi.Def.AbilityId == Ability_None {
		return false, nil
	}

	if abi.Status.State != AbiAct_Ready {
		return false, nil
	}

	_, err = tryActivate(e, &abi, targets, targetPos, ecsContainer)
	if err != nil {
		return false, fmt.Errorf("error trying to activate ability %v of entity %d: %v", abi.Def.AbilityId, e, err)
	}

	abiComp.abilities[abiIdx] = abi

	return true, nil
}

func (abilitiesManager) GetAbilityFunc(
	abilityId AbilityEnum,
) (AbilityFunc, error) {
	switch abilityId {
	case Ability_None:
		return nil, nil
	case Ability_Dodge:
		return DodgeAbility, nil
	case Ability_Dodge_Post:
		return DodgeAbilityPost, nil
	case Ability_Explode:
		return ExplodeAbility, nil
	case Ability_Spawn:
		return SpawnAbility, nil
	default:
		return nil, fmt.Errorf("ability function for ability %v not found", abilityId)
	}
}
