package collisionsystem

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"log"
	"slices"
)

func GetAABBCollisions(
	proximateEntities map[ecscommon.EntityId][]ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*components.PhysicsCollider,
	collisionLayers map[ecscommon.EntityId]*components.CollisionLayer,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) (map[ecscommon.EntityId][]ecscommon.EntityId, error) {
	collisions := make(map[ecscommon.EntityId][]ecscommon.EntityId)
	pcm := components.PhysicsColliderManager{}
	clm := components.CollisionLayersManager{}

	for eA, colEntities := range proximateEntities {
		aLayers, err := clm.GetLayers(eA, collisionLayers)
		if err != nil {
			log.Printf("Error getting collider layers for entity %d: %v\n", eA, err)
			continue
		}

		aMask, err := clm.GetMask(eA, collisionLayers)
		if err != nil {
			log.Printf("Error getting collider mask for entity %d: %v\n", eA, err)
			continue
		}

		aAABB, err := pcm.GetWorldPaddedAABB(eA, colliders, transforms, parents)
		if err != nil {
			log.Printf("Error getting AABB for entity %d: %v\n", eA, err)
			continue
		}

		for _, eB := range colEntities {
			if eA == eB {
				continue
			}

			bLayers, err := clm.GetLayers(eB, collisionLayers)
			if err != nil {
				log.Printf("Error getting collider layers for entity %d: %v\n", eB, err)
				continue
			}

			bMask, err := clm.GetMask(eB, collisionLayers)
			if err != nil {
				log.Printf("Error getting collider mask for entity %d: %v\n", eB, err)
				continue
			}

			if aLayers&bMask == 0 || bLayers&aMask == 0 {
				continue
			}

			if collidedEntities, ok := collisions[eB]; ok {
				if slices.Contains(collidedEntities, eA) {
					continue
				}
			}

			bAABB, err := pcm.GetWorldPaddedAABB(eB, colliders, transforms, parents)
			if err != nil {
				log.Printf("Error getting AABB for entity %d: %v\n", eB, err)
				continue
			}

			if utils.DetectAABBCollision(aAABB, bAABB) {
				v, ok := collisions[eA]
				if !ok {
					collisions[eA] = []ecscommon.EntityId{eB}
				}
				collisions[eA] = append(v, eB)
			}
		}
	}

	return collisions, nil
}
