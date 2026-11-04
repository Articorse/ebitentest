package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen"
)

// The center of the passable dry tile without a feature closest to near, going by generated terrain only, so it's meant for a new world
// that nothing has modified yet
func (cc *ChunkContainer) NewWorldSpawnPos(near utils.Vec2f) (utils.Vec2f, bool) {
	start := utils.WorldPosToTilePos(near)
	best, bestDistSq := utils.Vec2i{}, -1

	// Rings are squares, so a closer tile can still turn up in a later ring; the search ends once none can
	for ring := 0; ring <= data.WorldgenSpawnSearchRadiusTiles; ring++ {
		if bestDistSq != -1 && ring*ring > bestDistSq {
			break
		}

		for dy := -ring; dy <= ring; dy++ {
			for dx := -ring; dx <= ring; dx++ {
				if max(utils.AbsInt(dx), utils.AbsInt(dy)) != ring {
					continue
				}

				d := dx*dx + dy*dy
				if bestDistSq != -1 && d >= bestDistSq {
					continue
				}

				t := utils.Vec2i{X: start.X + dx, Y: start.Y + dy}
				tile := worldgen.Tile(cc.seed, t)
				def, ok := cc.atlas[tile]
				if ok && def.passable && !worldgen.IsWater(tile) && worldgen.Feature(cc.seed, t) == data.Feature_None {
					best, bestDistSq = t, d
				}
			}
		}
	}

	if bestDistSq == -1 {
		return near, false
	}
	return utils.TilePosToWorldPos(best), true
}
