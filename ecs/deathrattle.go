package ecs

import "ebittest/ecs/common"

type deathrattle struct {
	ability EntityAbility
}

func (x deathrattle) Copy() deathrattle {
	return deathrattle{
		ability: x.ability.copy(),
	}
}

func (x deathrattle) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Deathrattles.addComponent(e, x.Copy())
}

type deathrattleDto struct {
	Ability EntityAbility
}

func (x deathrattle) toDto() (ComponentDto, error) {
	return &deathrattleDto{
		Ability: x.ability.copy(),
	}, nil
}

func (x deathrattleDto) toComponent() (Component, error) {
	return &deathrattle{
		ability: x.Ability.copy(),
	}, nil
}
