package abilities

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/fov"
	"ebittest/utils"
	"fmt"
)

type TileStrikeParams struct {
	RangeTiles int
	Damage     int
	Mask       ecs.LayerMask
}

func (TileStrikeParams) IsAbilityParams() {}

func (p *TileStrikeParams) Copy() ecs.AbilityParams {
	if p == nil {
		return p
	}
	c := *p
	return &c
}

// Damages the tile at the target position, if it's in range, in sight and on a layer in the mask
func TileStrikeAbility(
	self common.EntityId,
	target ecs.AbilityTarget,
	params ecs.AbilityParams,
	ecsContainer *ecs.ECSContainer,
) (bool, error) {
	p, ok := params.(*TileStrikeParams)
	if !ok || p == nil {
		return false, fmt.Errorf("tile strike ability called with invalid params: %T", params)
	}

	// Measured from whoever holds the tool, since that's who has to reach and see the tile
	striker := self
	if holder := ecsContainer.Parent(self); holder != -1 {
		striker = holder
	}
	strikerPos, err := ecsContainer.WorldPos(striker)
	if err != nil {
		return false, fmt.Errorf("error getting world position of entity %d for tile strike: %v", striker, err)
	}

	origin := utils.WorldPosToTilePos(strikerPos)
	tile := utils.WorldPosToTilePos(target.Pos)
	terrain := ecsContainer.Terrain()

	if dx, dy := tile.X-origin.X, tile.Y-origin.Y; dx*dx+dy*dy > p.RangeTiles*p.RangeTiles {
		return false, nil
	}

	if layer, ok := terrain.TileLayer(tile); !ok || layer&p.Mask == 0 {
		return false, nil
	}

	inSight := false
	fov.Compute(origin, p.RangeTiles, terrain.IsOpaqueTile, func(t utils.Vec2i) {
		inSight = inSight || t == tile
	})
	if !inSight {
		return false, nil
	}

	return terrain.DamageTile(tile, p.Damage), nil
}
