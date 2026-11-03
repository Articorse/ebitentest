package abilities

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/fov"
	"ebittest/utils"
	"encoding/json"
	"fmt"
)

type PlaceTileParams struct {
	RangeTiles int
	Tile       data.TileEnum
}

func (PlaceTileParams) IsAbilityParams() {}

func (p *PlaceTileParams) Copy() ecs.AbilityParams {
	if p == nil {
		return p
	}
	c := *p
	return &c
}

func (p *PlaceTileParams) PlacedTile() data.TileEnum {
	return p.Tile
}

func decodePlaceTileParams(raw json.RawMessage) (ecs.AbilityParams, error) {
	var p PlaceTileParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("error unmarshalling place tile params: %w", err)
	}
	if p.RangeTiles <= 0 {
		return nil, fmt.Errorf("place tile params need a positive range, got %d", p.RangeTiles)
	}
	return &p, nil
}

// Only the tile under the aim, since placing on the first free tile along the way would put tiles where the player
// didn't point
func (p *PlaceTileParams) TargetTile(from, aim utils.Vec2f, ecsContainer *ecs.ECSContainer) (utils.Vec2i, bool) {
	if p == nil {
		return utils.Vec2i{}, false
	}

	aimed, ok := aimedTileInReach(from, aim, p.RangeTiles, ecsContainer.Terrain())
	if !ok || !ecsContainer.Terrain().CanPlaceTile(aimed) || ecsContainer.IsTileOccupied(aimed) {
		return utils.Vec2i{}, false
	}

	return aimed, true
}

func aimedTileInReach(from, aim utils.Vec2f, rangeTiles int, terrain ecs.Terrain) (utils.Vec2i, bool) {
	origin := utils.WorldPosToTilePos(from)
	aimed := utils.WorldPosToTilePos(aim)
	if utils.Dist2(origin, aimed) > rangeTiles*rangeTiles {
		return utils.Vec2i{}, false
	}

	inSight := false
	fov.Compute(origin, rangeTiles, terrain.IsOpaqueTile, func(t utils.Vec2i) {
		if t == aimed {
			inSight = true
		}
	})
	return aimed, inSight
}

func PlaceTileAbility(
	self common.EntityId,
	target ecs.AbilityTarget,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) (bool, error) {
	p, ok := params.(*PlaceTileParams)
	if !ok || p == nil {
		return false, fmt.Errorf("place tile ability called with invalid params: %T", params)
	}

	// Measured from whoever holds the item, since that's who has to reach and see the tile
	placer := self
	if holder := ecsContainer.Parent(self); holder != -1 {
		placer = holder
	}
	placerPos, err := ecsContainer.WorldPos(placer)
	if err != nil {
		return false, fmt.Errorf("error getting world position of entity %d for placing a tile: %v", placer, err)
	}

	if !target.HasPos {
		return false, nil
	}

	tile, ok := p.TargetTile(placerPos, target.Pos, ecsContainer)
	if !ok || !ecsContainer.Terrain().PlaceTile(tile, p.Tile) {
		return false, nil
	}

	if err := ecsContainer.ConsumeItem(self); err != nil {
		return true, fmt.Errorf("error using up item %d after placing a tile: %w", self, err)
	}

	return true, nil
}
