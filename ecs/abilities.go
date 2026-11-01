package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
)

type AbilityEnum uint64

// The other IDs are defined by the content that registers them
const Ability_None AbilityEnum = 0

type AbilityActivityEnum uint8

const (
	AbiAct_Disabled AbilityActivityEnum = iota
	AbiAct_Ready
	AbiAct_Active
	AbiAct_OnCooldown
)

type AbilityParams interface {
	IsAbilityParams()
	Copy() AbilityParams
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

func (x abilities) Copy() abilities {
	copyAbilityParams(x.abilities[:])
	return x
}

func (x abilities) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Abilities.addComponent(e, x.Copy())
}

func (x EntityAbility) copy() EntityAbility {
	if x.Params != nil {
		x.Params = x.Params.Copy()
	}
	return x
}

func copyAbilityParams(abis []EntityAbility) {
	for i, abi := range abis {
		abis[i] = abi.copy()
	}
}

type abilitiesDto struct {
	Abilities [data.MaxAbilitySlots]EntityAbility
}

func (x abilities) toDto() (ComponentDto, error) {
	copyAbilityParams(x.abilities[:])
	return &abilitiesDto{
		Abilities: x.abilities,
	}, nil
}

func (x abilitiesDto) toComponent() (Component, error) {
	copyAbilityParams(x.Abilities[:])
	return &abilities{
		abilities: x.Abilities,
	}, nil
}
