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

func NewAbilitiesComponent(defs map[AbilityEnum]AbilityDef) *abilities {
	abiMap := make(map[AbilityEnum]EntityAbility, len(defs))
	for k, v := range defs {
		abiMap[k] = EntityAbility{
			Def: v,
			Status: AbilityStatus{
				State: AbiAct_Ready,
			},
		}
	}

	return &abilities{abilities: abiMap}
}

func (AbilitiesManager) TickAbilities(e common.EntityId, world *World) error {
	abiComp, err := world.Abilities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	for _, a := range abiComp.abilities {
		switch a.Status.State {
		case AbiAct_Active:
			a.Status.DurationCounterMs -= 1000 / data.TPS

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
			a.Status.CooldownCounterMs -= 1000 / data.TPS

			if a.Status.CooldownCounterMs <= 0 {
				a.Status.State = AbiAct_Ready
			}
		}

		abiComp.abilities[a.Name] = a
	}

	return nil
}

func (AbilitiesManager) HasAbility(e common.EntityId, name AbilityEnum, world *World) bool {
	abiComp, err := world.Abilities.getComponent(e)
	if err != nil {
		return false
	}

	_, ok := abiComp.abilities[name]

	return ok
}

func (AbilitiesManager) DisableAbility(e common.EntityId, name AbilityEnum, world *World) error {
	abiComp, err := world.Abilities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	a, ok := abiComp.abilities[name]
	if !ok {
		return fmt.Errorf("entity %d does not have ability %v", e, name)
	}

	a.Status.State = AbiAct_Disabled
	abiComp.abilities[name] = a

	return nil
}

func (AbilitiesManager) EnableAbility(e common.EntityId, name AbilityEnum, world *World) error {
	abiComp, err := world.Abilities.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	a, ok := abiComp.abilities[name]
	if !ok {
		return fmt.Errorf("entity %d does not have ability %v", e, name)
	}

	a.Status.State = AbiAct_Ready
	abiComp.abilities[name] = a

	return nil
}

func (AbilitiesManager) ActivateAbility(
	e common.EntityId,
	targets []common.EntityId,
	name AbilityEnum,
	world *World,
) (activated bool, err error) {
	abiComp, err := world.Abilities.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get abilities component of entity %d: %v", e, err)
	}

	abi, ok := abiComp.abilities[name]
	if !ok {
		return false, fmt.Errorf("entity %d does not have ability %v", e, name)
	}

	if abi.Status.State != AbiAct_Ready {
		return false, nil
	}

	abi.Status.DurationCounterMs = abi.Def.DurationMs
	abi.Status.CooldownCounterMs = abi.Def.CooldownMs
	abi.Status.State = AbiAct_Active

	abiComp.abilities[name] = abi

	err = abi.Def.Effect(e, targets, world)
	if err != nil {
		return false, fmt.Errorf("error executing effect of ability %v of entity %d: %v", name, e, err)
	}

	return true, nil
}
