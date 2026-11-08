package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
)

// Whether the tile itself or its feature gives what's gathered on the harvest area
func (cc *ChunkContainer) IsHarvestable(t utils.Vec2i) bool {
	if _, def, ok := cc.featureOn(t); ok && def.harvestable {
		return true
	}
	tile, ok := cc.TileAt(t)
	return ok && cc.atlas[tile].harvestable
}

// Whether something harvestable drops the kind, which is what's gathered on the harvest area
func (cc *ChunkContainer) IsHarvestItem(kind data.ItemEnum) bool {
	for _, def := range cc.featureAtlas {
		if def.harvestable && def.drops.CanDrop(kind) {
			return true
		}
	}
	for _, def := range cc.atlas {
		if def.harvestable && def.drops.CanDrop(kind) {
			return true
		}
	}
	return false
}

func (cc *ChunkContainer) noteHarvested(t utils.Vec2i) {
	cc.harvested = append(cc.harvested, t)
}

// The tiles something harvestable has been felled, broken or picked up from since this was last asked, in the order it
// happened
func (cc *ChunkContainer) TakeHarvested() []utils.Vec2i {
	harvested := cc.harvested
	cc.harvested = nil
	return harvested
}
