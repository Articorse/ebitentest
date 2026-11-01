package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
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

type Abilities struct {
	abilities [data.MaxAbilitySlots]EntityAbility
}

func NewAbilityDef(id AbilityEnum, postId AbilityEnum, cooldownMs int, durationMs int) AbilityDef {
	return AbilityDef{
		AbilityId:     id,
		PostAbilityId: postId,
		CooldownMs:    cooldownMs,
		DurationMs:    durationMs,
	}
}

func NewAbilitiesComponent(defs [data.MaxAbilitySlots]EntityAbility) *Abilities {
	abis := [data.MaxAbilitySlots]EntityAbility{}

	for i, abi := range defs {
		abis[i] = EntityAbility{
			Def:    abi.Def,
			Status: AbilityStatus{State: AbiAct_Ready},
			Params: abi.Params,
		}
	}

	return &Abilities{
		abilities: abis,
	}
}

func (x *Abilities) Has(id AbilityEnum) bool {
	for _, a := range x.abilities {
		if a.Def.AbilityId == id {
			return true
		}
	}

	return false
}

func (x *Abilities) Disable(id AbilityEnum) error {
	return x.setState(id, AbiAct_Disabled)
}

func (x *Abilities) Enable(id AbilityEnum) error {
	return x.setState(id, AbiAct_Ready)
}

func (x *Abilities) setState(id AbilityEnum, state AbilityActivityEnum) error {
	for i, a := range x.abilities {
		if a.Def.AbilityId == id {
			x.abilities[i].Status.State = state
			return nil
		}
	}

	return fmt.Errorf("no ability %v", id)
}

func (x *ECSContainer) TickAbilities(e common.EntityId) error {
	abis, ok := x.Abilities.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no abilities", e)
	}

	for i, a := range abis.abilities {
		if err := tickAbilityState(e, &a, x); err != nil {
			return fmt.Errorf("error ticking ability %v of entity %d: %v", a.Def.AbilityId, e, err)
		}
		abis.abilities[i] = a
	}

	return nil
}

func (x *ECSContainer) ActivateAbility(
	e common.EntityId,
	targets []common.EntityId,
	targetPos utils.Vec2f,
	abiIdx int,
) (activated bool, err error) {
	if abiIdx > data.MaxAbilitySlots-1 {
		return false, fmt.Errorf("ability index %d is out of bounds", abiIdx)
	}

	abis, ok := x.Abilities.Get(e)
	if !ok {
		return false, fmt.Errorf("entity %d has no abilities", e)
	}

	abi := abis.abilities[abiIdx]

	if abi.Def.AbilityId == Ability_None || abi.Status.State != AbiAct_Ready {
		return false, nil
	}

	// Activated on a copy, so that a failed activation leaves the ability as it was
	if _, err := tryActivate(e, &abi, targets, targetPos, x); err != nil {
		return false, fmt.Errorf("error trying to activate ability %v of entity %d: %v", abi.Def.AbilityId, e, err)
	}
	abis.abilities[abiIdx] = abi

	return true, nil
}

func (x *ECSContainer) AbilityFunc(abilityId AbilityEnum) (AbilityFunc, error) {
	if abilityId == Ability_None {
		return nil, nil
	}

	abilityFunc, ok := x.abilityFuncs[abilityId]
	if !ok {
		return nil, fmt.Errorf("ability function for ability %v not found", abilityId)
	}

	return abilityFunc, nil
}

func (x Abilities) Copy() Abilities {
	copyAbilityParams(x.abilities[:])
	return x
}

func (x Abilities) addTo(ecsContainer *ECSContainer, e common.EntityId) {
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

func (x Abilities) toDto() (ComponentDto, error) {
	copyAbilityParams(x.abilities[:])
	return &abilitiesDto{
		Abilities: x.abilities,
	}, nil
}

func (x abilitiesDto) toComponent() (Component, error) {
	copyAbilityParams(x.Abilities[:])
	return &Abilities{
		abilities: x.Abilities,
	}, nil
}
