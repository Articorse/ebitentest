package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type Deathrattle struct {
	ability EntityAbility
}

func NewDeathrattleComponent(abi EntityAbility) (*Deathrattle, error) {
	return &Deathrattle{
		ability: abi,
	}, nil
}

func (x *ECSContainer) runDeathrattle(e common.EntityId) error {
	dr, ok := x.Deathrattles.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no deathrattle", e)
	}

	effect, err := x.AbilityFunc(dr.ability.Def.AbilityId)
	if err != nil {
		return fmt.Errorf("could not get ability function for ability %v of entity %d: %v", dr.ability.Def.AbilityId, e, err)
	}

	return effect(e, dr.ability.Params, x)
}

func (x *ECSContainer) TickDeathrattle(e common.EntityId) error {
	dr, ok := x.Deathrattles.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no deathrattle", e)
	}

	if err := tickAbilityState(e, &dr.ability, x); err != nil {
		return fmt.Errorf("error ticking ability %v of entity %d: %v", dr.ability.Def.AbilityId, e, err)
	}

	return nil
}

func (x Deathrattle) Copy() Deathrattle {
	return Deathrattle{
		ability: x.ability.copy(),
	}
}

func (x Deathrattle) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Deathrattles.addComponent(e, x.Copy())
}

type deathrattleDto struct {
	Ability EntityAbility
}

func (x Deathrattle) toDto() (ComponentDto, error) {
	return &deathrattleDto{
		Ability: x.ability.copy(),
	}, nil
}

func (x deathrattleDto) toComponent() (Component, error) {
	return &Deathrattle{
		ability: x.Ability.copy(),
	}, nil
}
