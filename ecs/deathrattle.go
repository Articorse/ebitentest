package ecs

type deathrattle struct {
	ability EntityAbility
}

func (deathrattle) isComponent() {}

func (x deathrattle) Copy() deathrattle {
	return deathrattle{
		ability: x.ability.copy(),
	}
}

type deathrattleDto struct {
	Ability EntityAbility
}

func (deathrattleDto) isComponentDto() {}

func (x deathrattle) ToDto() deathrattleDto {
	return deathrattleDto{
		Ability: x.ability.copy(),
	}
}

func (x deathrattleDto) ToComponent() *deathrattle {
	return &deathrattle{
		ability: x.Ability.copy(),
	}
}
