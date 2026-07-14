package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

func tickAbilityState(owner common.EntityId, abi *EntityAbility, ecsContainer *ECSContainer) error {
	switch abi.Status.State {
	case AbiAct_Active:
		abi.Status.DurationCounterMs -= data.TickMs

		if abi.Status.DurationCounterMs <= 0 {
			abi.Status.State = AbiAct_OnCooldown

			postEffect, err := ecsContainer.AbilitiesManager.GetAbilityFunc(abi.Def.AbilityId)
			if err != nil {
				return fmt.Errorf("error getting post effect of ability %v of entity %d: %v", abi.Def.AbilityId, owner, err)
			}

			if postEffect != nil {
				err := postEffect(owner, abi.Params, ecsContainer)
				if err != nil {
					return fmt.Errorf("error executing post effect of ability %v of entity %d: %v", abi.Def.AbilityId, owner, err)
				}
			}
		}
	case AbiAct_OnCooldown:
		abi.Status.CooldownCounterMs -= data.TickMs

		if abi.Status.CooldownCounterMs <= 0 {
			abi.Status.State = AbiAct_Ready
		}
	}

	return nil
}

func tryActivate(
	owner common.EntityId,
	abi *EntityAbility,
	targets []common.EntityId,
	targetPos utils.Vec2f,
	ecsContainer *ECSContainer,
) (bool, error) {
	if abi.Def.AbilityId == Ability_None {
		return false, nil
	}

	if abi.Status.State != AbiAct_Ready {
		return false, nil
	}

	abi.Status.DurationCounterMs = abi.Def.DurationMs
	abi.Status.CooldownCounterMs = abi.Def.CooldownMs
	abi.Status.State = AbiAct_Active

	effect, err := ecsContainer.AbilitiesManager.GetAbilityFunc(abi.Def.AbilityId)
	if err != nil {
		return false, fmt.Errorf("error getting effect of ability %v of entity %d: %v", abi.Def.AbilityId, owner, err)
	}

	err = effect(owner, abi.Params, ecsContainer)
	if err != nil {
		return false, fmt.Errorf("error executing effect of ability %v of entity %d: %v", abi.Def.AbilityId, owner, err)
	}

	return true, nil
}
