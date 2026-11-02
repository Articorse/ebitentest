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

// The tile under the aim if the strike can affect it. Otherwise the last terrain tile between from and aim that it can
// affect, so that aiming past a wall, such as at something behind it, hits the wall. Ground is only hit when aimed at
// directly, since the ground in front of where the player aims gets in the way of every strike, and destroying it
// moves the target closer with each one.
func (p *TileStrikeParams) TargetTile(from, aim utils.Vec2f, terrain ecs.Terrain) (utils.Vec2i, bool) {
	if p == nil {
		return utils.Vec2i{}, false
	}

	origin := utils.WorldPosToTilePos(from)
	inSight := map[utils.Vec2i]bool{}
	fov.Compute(origin, p.RangeTiles, terrain.IsOpaqueTile, func(t utils.Vec2i) {
		inSight[t] = true
	})

	if aimed := utils.WorldPosToTilePos(aim); p.canAffect(origin, aimed, inSight, terrain) {
		return aimed, true
	}

	target, found := utils.Vec2i{}, false
	for _, t := range utils.TilesAlongSegment(from, aim) {
		if layer, _ := terrain.TileLayer(t); layer == ecs.Layer_Terrain && p.canAffect(origin, t, inSight, terrain) {
			target, found = t, true
		}
	}

	return target, found
}

func (p *TileStrikeParams) canAffect(origin, tile utils.Vec2i, inSight map[utils.Vec2i]bool, terrain ecs.Terrain) bool {
	if utils.Dist2(origin, tile) > p.RangeTiles*p.RangeTiles || !inSight[tile] {
		return false
	}

	layer, ok := terrain.TileLayer(tile)
	return ok && layer&p.Mask != 0 && terrain.CanDamageTile(tile)
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

	if !target.HasPos {
		return false, nil
	}

	terrain := ecsContainer.Terrain()
	tile, ok := p.TargetTile(strikerPos, target.Pos, terrain)
	if !ok || !terrain.DamageTile(tile, p.Damage) {
		return false, nil
	}

	// The strike already landed, so a missing damage number mustn't undo its cooldown
	if err := effects.SpawnDamageNumber(utils.TilePosToWorldPos(tile), p.Damage, ecsContainer); err != nil {
		log.Println(err)
	}

	return true, nil
}
