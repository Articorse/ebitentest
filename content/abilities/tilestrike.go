package abilities

import (
	"ebittest/content/effects"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/fov"
	"ebittest/utils"
	"fmt"
	"log"
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

// In range, in sight, on a layer in the mask and damageable
func (p *TileStrikeParams) CanTargetTile(from, tile utils.Vec2i, terrain ecs.Terrain) bool {
	if p == nil {
		return false
	}

	if dx, dy := tile.X-from.X, tile.Y-from.Y; dx*dx+dy*dy > p.RangeTiles*p.RangeTiles {
		return false
	}

	if layer, ok := terrain.TileLayer(tile); !ok || layer&p.Mask == 0 || !terrain.CanDamageTile(tile) {
		return false
	}

	inSight := false
	fov.Compute(from, p.RangeTiles, terrain.IsOpaqueTile, func(t utils.Vec2i) {
		inSight = inSight || t == tile
	})
	return inSight
}

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

	tile := utils.WorldPosToTilePos(target.Pos)
	terrain := ecsContainer.Terrain()
	if !p.CanTargetTile(utils.WorldPosToTilePos(strikerPos), tile, terrain) || !terrain.DamageTile(tile, p.Damage) {
		return false, nil
	}

	// The strike already landed, so a missing damage number mustn't undo its cooldown
	if err := effects.SpawnDamageNumber(utils.TilePosToWorldPos(tile), p.Damage, ecsContainer); err != nil {
		log.Println(err)
	}

	return true, nil
}
