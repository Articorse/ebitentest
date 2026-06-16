package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
)

func tickAbilityState(owner common.EntityId, abi *EntityAbility, world *World) error {
	switch abi.Status.State {
	case AbiAct_Active:
		abi.Status.DurationCounterMs -= data.TickMs

		if abi.Status.DurationCounterMs <= 0 {
			abi.Status.State = AbiAct_OnCooldown

			if abi.Def.PostEffect != nil {
				err := abi.Def.PostEffect(owner, nil, world)
				if err != nil {
					return fmt.Errorf("error executing post effect of ability %v of entity %d: %v", abi.Name, owner, err)
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

func tryActivate(owner common.EntityId, abi *EntityAbility, targets []common.EntityId, world *World) (bool, error) {
	if abi.Name == Ability_None {
		return false, nil
	}

	if abi.Status.State != AbiAct_Ready {
		return false, nil
	}

	abi.Status.DurationCounterMs = abi.Def.DurationMs
	abi.Status.CooldownCounterMs = abi.Def.CooldownMs
	abi.Status.State = AbiAct_Active

	err := abi.Def.Effect(owner, targets, world)
	if err != nil {
		return false, fmt.Errorf("error executing effect of ability %v of entity %d: %v", abi.Name, owner, err)
	}

	return true, nil
}
