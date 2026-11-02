package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
)

// Params of abilities aimed at a single tile, so that whether they'd land can be shown before they're used
type TileTargeting interface {
	CanTargetTile(from, tile utils.Vec2i, terrain Terrain) bool
}

// The first tile-targeting ability of what the entity holds in its main hand
func (x *ECSContainer) HeldTileTargeting(e common.EntityId) (TileTargeting, bool) {
	equipper, ok := x.Equippers.Get(e)
	if !ok {
		return nil, false
	}

	item, ok := equipper.InSlot(Equip_MainHand)
	if !ok || x.IsDisabled(item) {
		return nil, false
	}

	eq, ok := x.Equipments.Get(item)
	if !ok {
		return nil, false
	}

	for _, abi := range eq.abilities {
		if targeting, ok := abi.Params.(TileTargeting); ok {
			return targeting, true
		}
	}

	return nil, false
}
