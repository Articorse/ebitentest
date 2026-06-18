package ecs

type deathrattle struct {
	ability EntityAbility
}

func (deathrattle) isComponent() {}

func (x deathrattle) Copy() deathrattle {
	return deathrattle{
		ability: x.ability,
	}
}
