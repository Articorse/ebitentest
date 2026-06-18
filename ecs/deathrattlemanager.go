package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type deathrattleManager struct{}

func NewDeathrattleComponent(abi EntityAbility) (*deathrattle, error) {
	return &deathrattle{
		ability: abi,
	}, nil
}

func (deathrattleManager) Effect(e common.EntityId, world *World) error {
	dr, err := world.Deathrattles.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get deathrattle component of entity %d: %v", e, err)
	}

	return dr.ability.Def.Effect(e, nil, utils.Vec2{}, world)
}

func (deathrattleManager) TickAbilities(e common.EntityId, world *World) error {
	drComp, err := world.Deathrattles.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get deathrattle component of entity %d: %v", e, err)
	}

	err = tickAbilityState(e, &drComp.ability, world)
	if err != nil {
		return fmt.Errorf("error ticking ability %v of entity %d: %v", drComp.ability.Name, e, err)
	}

	return nil
}
