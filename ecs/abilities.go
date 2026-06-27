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
	Ability_Dodge_Post
	Ability_Explode
)

type AbilityActivityEnum uint8

const (
	AbiAct_Disabled AbilityActivityEnum = iota
	AbiAct_Ready
	AbiAct_Active
	AbiAct_OnCooldown
)

type AbilityParams interface {
	IsAbilityParams()
}

type AbilityFunc func(
	self common.EntityId,
	params AbilityParams,
	ecsContainer *ECSContainer,
) error

type AbilityDef struct {
	AbilityId     AbilityEnum
	PostAbilityId AbilityEnum
	CooldownMs    int
	DurationMs    int
}

type AbilityStatus struct {
	CooldownCounterMs int
	DurationCounterMs int
	State             AbilityActivityEnum
}

type EntityAbility struct {
	Def    AbilityDef
	Params AbilityParams
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
			Def:    abi.Def,
			Status: abi.Status,
			Params: abi.Params,
		}
	}

	return abisCopy
}

type abilitiesDto struct {
	Abilities [data.MaxAbilitySlots]EntityAbility
}

func (abilitiesDto) isComponentDto() {}

func (x abilities) ToDto() abilitiesDto {
	return abilitiesDto{
		Abilities: x.abilities,
	}
}

func (x abilitiesDto) ToComponent() *abilities {
	return &abilities{
		abilities: x.Abilities,
	}
}
