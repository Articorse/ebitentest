package abilities

import (
	"ebittest/content/effects"
	"ebittest/ecs"
	"ebittest/ecs/common"
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

// The tile under the aim if the strike can affect it. Otherwise the last terrain tile, or tree, between from and aim that
// it can affect, so that aiming past a wall, such as at something behind it, hits the wall. Ground is only hit when aimed at
// directly, since the ground in front of where the player aims gets in the way of every strike, and destroying it
// moves the target closer with each one.
func (p *TileStrikeParams) TargetTile(holder common.EntityId, aim utils.Vec2f, ecsContainer *ecs.ECSContainer) (utils.Vec2i, bool) {
	if p == nil {
		return utils.Vec2i{}, false
	}
	from, err := ecsContainer.WorldPos(holder)
	if err != nil {
		return utils.Vec2i{}, false
	}
	body := ecsContainer.BodyTiles(holder, utils.WorldPosToTilePos(from))

	if aimed := utils.WorldPosToTilePos(aim); p.canAffect(body, aimed, ecsContainer) {
		return aimed, true
	}

	target, found := utils.Vec2i{}, false
	for _, t := range utils.TilesAlongSegment(from, aim) {
		if layer, _ := struckLayer(t, ecsContainer.Terrain()); layer&(ecs.Layer_Wall|ecs.Layer_Feature) != 0 && p.canAffect(body, t, ecsContainer) {
			target, found = t, true
		}
	}

	return target, found
}

func (p *TileStrikeParams) canAffect(body []utils.Vec2i, tile utils.Vec2i, ecsContainer *ecs.ECSContainer) bool {
	if !ecsContainer.TileInReachFrom(body, tile, p.RangeTiles) {
		return false
	}
	terrain := ecsContainer.Terrain()

	layer, ok := struckLayer(tile, terrain)
	if !ok || layer&p.Mask == 0 {
		return false
	}
	return terrain.CanDamageFeature(tile) || terrain.CanDamageTile(tile)
}

// A feature that can be damaged takes strikes aimed at its tile instead of the ground under it, on the layer its
// collider has. One that can't, such as a stump, doesn't get in the way.
func struckLayer(tile utils.Vec2i, terrain ecs.Terrain) (ecs.LayerMask, bool) {
	if terrain.CanDamageFeature(tile) {
		return ecs.Layer_Feature, true
	}
	return terrain.TileCollisionLayer(tile)
}

func strikeTile(tile utils.Vec2i, damage int, ecsContainer *ecs.ECSContainer) bool {
	if ecsContainer.Terrain().CanDamageFeature(tile) {
		return ecsContainer.DamageFeature(tile, damage)
	}
	return ecsContainer.DamageTile(tile, damage)
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
	if !target.HasPos {
		return false, nil
	}

	tile, ok := p.TargetTile(striker, target.Pos, ecsContainer)
	if !ok || !strikeTile(tile, p.Damage, ecsContainer) {
		return false, nil
	}

	// The strike already landed, so a missing damage number mustn't undo its cooldown
	if err := effects.SpawnDamageNumber(utils.TilePosToWorldPos(tile), p.Damage, ecsContainer); err != nil {
		log.Println(err)
	}

	return true, nil
}
