package abilities

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/json"
	"fmt"
)

type PlaceFeatureParams struct {
	RangeTiles int
	Feature    data.FeatureEnum
}

func (PlaceFeatureParams) IsAbilityParams() {}

func (p *PlaceFeatureParams) Copy() ecs.AbilityParams {
	if p == nil {
		return p
	}
	c := *p
	return &c
}

func (p *PlaceFeatureParams) PlacedFeature() data.FeatureEnum {
	return p.Feature
}

func decodePlaceFeatureParams(raw json.RawMessage) (ecs.AbilityParams, error) {
	var p PlaceFeatureParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("error unmarshalling place feature params: %w", err)
	}
	if p.RangeTiles <= 0 {
		return nil, fmt.Errorf("place feature params need a positive range, got %d", p.RangeTiles)
	}
	return &p, nil
}

// Only the tile under the aim, like placing a tile. Unlike a tile, it may go where something stands, so features placed
// this way should be walked through.
func (p *PlaceFeatureParams) TargetTile(holder common.EntityId, aim utils.Vec2f, ecsContainer *ecs.ECSContainer) (utils.Vec2i, bool) {
	if p == nil {
		return utils.Vec2i{}, false
	}

	aimed, ok := ecsContainer.AimedTileInReach(holder, aim, p.RangeTiles)
	if !ok || !ecsContainer.Terrain().CanPlaceFeature(aimed) {
		return utils.Vec2i{}, false
	}

	return aimed, true
}

func PlaceFeatureAbility(
	self common.EntityId,
	target ecs.AbilityTarget,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) (bool, error) {
	p, ok := params.(*PlaceFeatureParams)
	if !ok || p == nil {
		return false, fmt.Errorf("place feature ability called with invalid params: %T", params)
	}

	placer := self
	if holder := ecsContainer.Parent(self); holder != -1 {
		placer = holder
	}
	if !target.HasPos {
		return false, nil
	}

	tile, ok := p.TargetTile(placer, target.Pos, ecsContainer)
	if !ok || !ecsContainer.Terrain().PlaceFeature(tile, p.Feature) {
		return false, nil
	}

	if err := ecsContainer.ConsumeItem(self); err != nil {
		return true, fmt.Errorf("error using up item %d after placing a feature: %w", self, err)
	}

	return true, nil
}
