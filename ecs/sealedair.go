package ecs

import (
	"ebittest/climate"
	"ebittest/data"
	"ebittest/ecs/common"
	"maps"
	"slices"
)

// Air sealed inside something, such as a spacesuit, that nothing gets into or out of. Whoever wears it breathes it
// rather than the air around them, and feels neither the cold nor the heat outside.
type SealedAir struct {
	// In milligrams in all, not per tile
	air climate.Air
	// How many tiles' worth of air it holds, which is what the air in it is spread over when it's breathed
	tiles int64
}

func NewSealedAirComponent(air climate.Air, tiles int64) *SealedAir {
	return &SealedAir{air: air, tiles: max(tiles, 1)}
}

func (x *SealedAir) Contents() climate.Air {
	return x.air
}

func (x *SealedAir) PerTile() climate.Air {
	perTile := climate.Air{}
	for g, mg := range x.air {
		perTile[g] = mg / x.tiles
	}
	return perTile
}

// What the entity wears that has air sealed in it, if anything
func (x *ECSContainer) WornSealedAir(e common.EntityId) (*SealedAir, bool) {
	equipper, ok := x.Equippers.Get(e)
	if !ok {
		return nil, false
	}
	for _, slot := range slices.Sorted(maps.Keys(equipper.equipment)) {
		if slot == Equip_MainHand {
			continue
		}
		if sealed, ok := x.SealedAirs.Get(equipper.equipment[slot]); ok {
			return sealed, true
		}
	}
	return nil, false
}

// Breathes in up to the oxygen given from what the entity wears that has air sealed in it, and as much carbon dioxide
// out into it. Whether it breathes sealed air.
func (x *ECSContainer) BreatheSealedAir(e common.EntityId, oxygenMg int64) bool {
	sealed, ok := x.WornSealedAir(e)
	if !ok {
		return false
	}
	mg := min(sealed.air[data.Gas_Oxygen], oxygenMg)
	sealed.air[data.Gas_Oxygen] -= mg
	sealed.air[data.Gas_CarbonDioxide] += mg
	return true
}

// The temperature of the air around the entity, or inside what it wears that has air sealed in it
func (x *ECSContainer) AirTemperatureOf(e common.EntityId) (climate.Centicelsius, error) {
	if _, ok := x.WornSealedAir(e); ok {
		return data.SealedAirCelsius * 100, nil
	}
	pos, err := x.WorldPos(e)
	if err != nil {
		return 0, err
	}
	return x.terrain.AirTemperature(x.AirTile(pos)), nil
}

func (x SealedAir) Copy() SealedAir {
	return x
}

func (x SealedAir) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.SealedAirs.addComponent(e, x.Copy())
}

type sealedAirDto struct {
	Air   climate.Air
	Tiles int64
}

func (x SealedAir) toDto() (ComponentDto, error) {
	return &sealedAirDto{Air: x.air, Tiles: x.tiles}, nil
}

func (x sealedAirDto) toComponent() (Component, error) {
	return NewSealedAirComponent(x.Air, x.Tiles), nil
}
