package ecs

import "ebittest/ecs/common"

type AbilityEnum uint64

type AbilityActivityEnum uint8

const (
	AbiAct_Disabled AbilityActivityEnum = iota
	AbiAct_Ready
	AbiAct_Active
	AbiAct_OnCooldown
)

type AbilityFunc func(
	self common.EntityId,
	targets []common.EntityId,
	world *World,
) error

type AbilityDef struct {
	Effect     AbilityFunc
	CooldownMs int
	DurationMs int
	PostEffect AbilityFunc
}

type AbilityStatus struct {
	CooldownCounterMs int
	DurationCounterMs int
	State             AbilityActivityEnum
}

type EntityAbility struct {
	Name   AbilityEnum
	Def    AbilityDef
	Status AbilityStatus
}

type abilities struct {
	abilities map[AbilityEnum]EntityAbility
}

func (abilities) isComponent() {}

func (x abilities) Copy() abilities {
	abilitiesCopy := make(map[AbilityEnum]EntityAbility, len(x.abilities))
	for k, v := range x.abilities {
		abilitiesCopy[k] = v
	}

	return abilities{abilities: abilitiesCopy}
}
