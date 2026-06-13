package platformsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

func Tick(
	shg map[common.CellKey][]common.EntityId,
	world *ecs.World,
) error {
	tm := ecs.TransformManager{}
	pcm := ecs.PlatformColliderManager{}
	pm := ecs.ParentManager{}
	phcm := ecs.PhysicsColliderManager{}

	for _, eA := range world.PlatformColliders.GetEntities() {
		aAABB, err := pcm.GetWorldAABB(eA, world)
		if err != nil {
			log.Printf("error getting AABB of entity %d: %v", eA, err)
			continue
		}

		aWorldPos, err := tm.GetWorldPos(eA, world)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", eA, err)
			continue
		}

		aLayer, err := pcm.GetLayer(eA, world)
		if err != nil {
			log.Printf("error getting layer of entity %d: %v", eA, err)
			continue
		}

		aMask, err := pcm.GetMask(eA, world)
		if err != nil {
			log.Printf("error getting mask of entity %d: %v", eA, err)
			continue
		}

		startCellX := int(aWorldPos.X / data.SpatialHashGridCellSize)
		startCellY := int(aWorldPos.Y / data.SpatialHashGridCellSize)

		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for _, eB := range shg[common.CellKey{X: startCellX + dx, Y: startCellY + dy}] {
					if eA == eB {
						continue
					}

					if !world.PhysicsColliders.HasComponent(eB) {
						continue
					}

					bLayer, err := phcm.GetLayer(eB, world)
					if err != nil {
						log.Printf("error getting layer of entity %d: %v", eB, err)
						continue
					}

					bMask, err := phcm.GetMask(eB, world)
					if err != nil {
						log.Printf("error getting mask of entity %d: %v", eB, err)
						continue
					}

					if (aLayer&bMask) == 0 || (bLayer&aMask) == 0 {
						continue
					}

					bWorldPos, err := tm.GetWorldPos(eB, world)
					if err != nil {
						log.Printf("error getting world position of entity %d: %v", eB, err)
						continue
					}

					if utils.PointInAABB(bWorldPos, aAABB) {
						err := pm.Attach(eB, eA, world)
						if err != nil {
							log.Printf("error attaching entity %d to platform entity %d: %v", eB, eA, err)
						}
						continue
					}

					pEnt := pm.GetEntity(eB, world)

					if pEnt == eA {
						err := pm.Detach(eB, world)
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
