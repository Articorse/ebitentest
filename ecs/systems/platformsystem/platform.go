package platformsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"log"
)

func Tick(
	shg map[ecscommon.CellKey][]ecscommon.EntityId,
	platforms map[ecscommon.EntityId]*components.Platform,
	transforms map[ecscommon.EntityId]*components.Transform,
	collisionLayers map[ecscommon.EntityId]*components.CollisionLayer,
	colliders map[ecscommon.EntityId]*components.PlatformCollider,
	parents map[ecscommon.EntityId]*components.Parent,
) error {
	tm := components.TransformManager{}
	pcm := components.PlatformColliderManager{}
	clm := components.CollisionLayersManager{}
	pm := components.ParentManager{}

	for eA, _ := range platforms {
		aAABB, err := pcm.GetWorldAABB(eA, colliders, transforms, parents)
		if err != nil {
			log.Printf("error getting AABB of entity %d: %v", eA, err)
			continue
		}

		aWorldPos, err := tm.GetWorldPos(eA, transforms, parents)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", eA, err)
			continue
		}

		aLayers, err := clm.GetLayers(eA, collisionLayers)
		if err != nil {
			log.Printf("error getting layers of entity %d: %v", eA, err)
			continue
		}

		aMask, err := clm.GetMask(eA, collisionLayers)
		if err != nil {
			log.Printf("error getting mask of entity %d: %v", eA, err)
			continue
		}

		startCellX := int(aWorldPos.X / data.SpatialHashGridCellSize)
		startCellY := int(aWorldPos.Y / data.SpatialHashGridCellSize)

		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for _, eB := range shg[ecscommon.CellKey{X: startCellX + dx, Y: startCellY + dy}] {
					if eA == eB {
						continue
					}

					_, ok := collisionLayers[eB]
					if !ok {
						continue
					}

					bLayers, err := clm.GetLayers(eB, collisionLayers)
					if err != nil {
						log.Printf("error getting layers of entity %d: %v", eB, err)
						continue
					}

					bMask, err := clm.GetMask(eB, collisionLayers)
					if err != nil {
						log.Printf("error getting mask of entity %d: %v", eB, err)
						continue
					}

					if (aLayers&bMask) == 0 || (bLayers&aMask) == 0 {
						continue
					}

					bWorldPos, err := tm.GetWorldPos(eB, transforms, parents)
					if err != nil {
						log.Printf("error getting world position of entity %d: %v", eB, err)
						continue
					}

					if utils.PointInAABB(bWorldPos, aAABB) {
						err := pm.Attach(eB, eA, transforms, parents)
						if err != nil {
							log.Printf("error attaching entity %d to platform entity %d: %v", eB, eA, err)
						}
						continue
					}

					pEnt := pm.GetEntity(eB, parents)

					if pEnt == eA {
						err := pm.Detach(eB, transforms, parents)
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
