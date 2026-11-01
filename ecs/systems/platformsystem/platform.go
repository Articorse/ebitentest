package platformsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

func Tick(
	shg map[utils.Vec2i][]common.EntityId,
	ecsContainer *ecs.ECSContainer,
) error {
	for eA, aCol := range ecs.Query1(&ecsContainer.PlatformColliders) {
		aAABB, err := ecsContainer.WorldAABB(eA, aCol.LocalAABB())
		if err != nil {
			log.Printf("error getting AABB of entity %d: %v", eA, err)
			continue
		}

		aWorldPos, err := ecsContainer.WorldPos(eA)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", eA, err)
			continue
		}

		aLayer, aMask := aCol.Layer(), aCol.Mask()

		startCellX := int(aWorldPos.X / data.SpatialHashGridCellSize)
		startCellY := int(aWorldPos.Y / data.SpatialHashGridCellSize)

		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for _, eB := range shg[utils.Vec2i{X: startCellX + dx, Y: startCellY + dy}] {
					if eA == eB {
						continue
					}

					bCol, ok := ecsContainer.PhysicsColliders.Get(eB)
					if !ok {
						continue
					}

					if (aLayer&bCol.Mask()) == 0 || (bCol.Layer()&aMask) == 0 {
						continue
					}

					bWorldPos, err := ecsContainer.WorldPos(eB)
					if err != nil {
						log.Printf("error getting world position of entity %d: %v", eB, err)
						continue
					}

					if utils.PointInAABB(bWorldPos, aAABB) {
						err := ecsContainer.Attach(eB, eA, ecs.Link_Attached)
						if err != nil {
							log.Printf("error attaching entity %d to platform entity %d: %v", eB, eA, err)
						}
						continue
					}

					pEnt := ecsContainer.Parent(eB)

					if pEnt == eA {
						err := ecsContainer.Detach(eB)
						if err != nil {
							log.Printf("error detaching entity %d to platform entity %d: %v", eB, eA, err)
						}
					}
				}
			}
		}
	}

	return nil
}
