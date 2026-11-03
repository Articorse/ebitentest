package visionsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"log"
)

func Tick(ecsContainer *ecs.ECSContainer) {
	for e, v := range ecs.Query1(&ecsContainer.Visions, ecs.Without(&ecsContainer.Disabled)) {
		pos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("error getting world position of entity %d for its vision: %v", e, err)
			continue
		}

		origin := utils.WorldPosToTilePos(pos)
		rangeTiles := v.BaseRangeTiles() * ecsContainer.Capacity(e, data.Capacity_Sight) / data.FullEfficiency
		light := v.Light()
		ecsContainer.LightAround(origin, rangeTiles, light)
		v.Update(origin, rangeTiles, ecsContainer.Terrain().IsOpaqueTile, light.At)
	}
}
