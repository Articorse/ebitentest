package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
)

type AbilityEnum uint64

const (
	Ability_None AbilityEnum = iota
	Ability_Spawn
	Ability_Dodge
)

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
	abilities [data.MaxAbilitySlots]EntityAbility
}

func (abilities) isComponent() {}

func (x abilities) Copy() abilities {
	abisCopy := abilities{}

	for i, abi := range x.abilities {
		abisCopy.abilities[i] = EntityAbility{
			Name:   abi.Name,
			Def:    abi.Def,
			Status: abi.Status,
		}
	}

	return abisCopy
}
