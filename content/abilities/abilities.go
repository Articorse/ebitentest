package abilities

import (
	"ebittest/ecs"
	"encoding/gob"
	"fmt"
)

// Saved components refer to abilities by these values, so they must never change
const (
	Ability_Spawn      ecs.AbilityEnum = 1
	Ability_Dodge      ecs.AbilityEnum = 2
	Ability_Dodge_Post ecs.AbilityEnum = 3
	Ability_Explode    ecs.AbilityEnum = 4
	Ability_TileStrike ecs.AbilityEnum = 5
	Ability_PlaceTile  ecs.AbilityEnum = 6
	Ability_Slash      ecs.AbilityEnum = 7
)

func Register(ecsContainer *ecs.ECSContainer) error {
	gob.Register(&DodgeParams{})
	gob.Register(&ExplodeParams{})
	gob.Register(&TileStrikeParams{})
	gob.Register(&PlaceTileParams{})
	gob.Register(&SlashParams{})

	abilityFuncs := []struct {
		id          ecs.AbilityEnum
		abilityFunc ecs.AbilityFunc
	}{
		{Ability_Spawn, SpawnAbility},
		{Ability_Dodge, DodgeAbility},
		{Ability_Dodge_Post, DodgeAbilityPost},
		{Ability_Explode, ExplodeAbility},
		{Ability_TileStrike, TileStrikeAbility},
		{Ability_PlaceTile, PlaceTileAbility},
		{Ability_Slash, SlashAbility},
	}

	for _, a := range abilityFuncs {
		if err := ecsContainer.RegisterAbility(a.id, a.abilityFunc); err != nil {
			return fmt.Errorf("error registering ability %v: %w", a.id, err)
		}
	}

	if err := ecsContainer.RegisterAbilityParamsDecoder(Ability_PlaceTile, decodePlaceTileParams); err != nil {
		return fmt.Errorf("error registering the params decoder of ability %v: %w", Ability_PlaceTile, err)
	}

	return nil
}
