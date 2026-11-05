package visionsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"log"
)

func Tick(ecsContainer *ecs.ECSContainer) {
	for e, v := range ecs.Query1(&ecsContainer.Visions, ecs.Without(&ecsContainer.Disabled)) {
		if !v.IsDue(e, ecsContainer.TickIdx) {
			continue
		}
		pos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("error getting world position of entity %d for its vision: %v", e, err)
			continue
		}

		origin := utils.WorldPosToTilePos(pos)
		rangeTiles := v.BaseRangeTiles() * ecsContainer.Capacity(e, data.Capacity_Sight) / data.FullEfficiency
		light := v.Light()
		terrain := ecsContainer.Terrain()
		// What's seen only depends on what the light does: the terrain, and where from and how far
		if ecsContainer.LightAround(origin, rangeTiles, light) {
			v.Update(origin, rangeTiles, func(t utils.Vec2i) bool { return light.IsOpaque(t, terrain) }, light.At)
		}
	}
}
