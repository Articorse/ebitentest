package visionsystem

import (
	"ebittest/ecs"
	"ebittest/utils"
	"log"
)

func Tick(ecsContainer *ecs.ECSContainer) {
	darkness := ecs.Darkness(ecsContainer.TickIdx)
	for e, v := range ecs.Query1(&ecsContainer.Visions, ecs.Without(&ecsContainer.Disabled)) {
		pos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("error getting world position of entity %d for its vision: %v", e, err)
			continue
		}

		v.Update(utils.WorldPosToTilePos(pos), ecs.VisionReach(v.Radius(), darkness), ecsContainer.Terrain().IsOpaqueTile)
	}
}
