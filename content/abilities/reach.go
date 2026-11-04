package abilities

import (
	"ebittest/ecs"
	"ebittest/fov"
	"ebittest/utils"
)

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
