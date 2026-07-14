package collisionsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"log"
	"slices"
)

func GetCollisions(
	aColManager ecs.IColliderManager,
	bColManager ecs.IColliderManager,
	potentialCollisions map[common.EntityId][]common.EntityId,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId]map[common.EntityId]common.Collision, error) {
	collisions := make(map[common.EntityId]map[common.EntityId]common.Collision)
	for e1, colEntities := range potentialCollisions {
		for _, e2 := range colEntities {
			if e1 == e2 {
				continue
			}

			e1HasA := aColManager.HasCollider(e1, ecsContainer)
			e1HasB := bColManager.HasCollider(e1, ecsContainer)
			e2HasA := aColManager.HasCollider(e2, ecsContainer)
			e2HasB := bColManager.HasCollider(e2, ecsContainer)

			eA := common.EntityId(-1)
			eB := common.EntityId(-1)

			if e1HasA && e2HasB {
				eA = e1
				eB = e2
			} else if e1HasB && e2HasA {
				eA = e2
				eB = e1
			}

			if eA == -1 || eB == -1 {
				continue
			}

			if collidedEntities, ok := collisions[eB]; ok {
				if _, ok := collidedEntities[eA]; ok {
					continue
				}
			}

			aColShapes, err := aColManager.GetShapes(eA, ecsContainer)
			if err != nil {
				log.Printf("Error getting collider shapes for entity %d: %v\n", eA, err)
				continue
			}

			bColShapes, err := bColManager.GetShapes(eB, ecsContainer)
			if err != nil {
				log.Printf("Error getting collider shapes for entity %d: %v\n", eB, err)
				continue
			}

			collisionFound := false
			var collisionVector utils.Vec2f
			var aCollidedIdx int
			var bCollidedIdx int

			for aColShapeIdx, aColShape := range aColShapes {
				if collisionFound {
					break
				}

				for bColShapeIdx, bColShape := range bColShapes {
					if collisionFound {
						break
					}

					aCollidedIdx = aColShapeIdx
					bCollidedIdx = bColShapeIdx

					switch aS := aColShape.(type) {
					case *shapes.RectangleShape:
						switch bS := bColShape.(type) {
						case *shapes.RectangleShape:
							collisionVector = GetRectangleRectangleCollision(eA, eB, *aS, *bS, ecsContainer)
							if !collisionVector.IsZero() {
								collisionFound = true
							}
						case *shapes.CircleShape:
							collisionVector = GetRectangleCircleCollision(eA, eB, *aS, *bS, ecsContainer)
							if !collisionVector.IsZero() {
								collisionFound = true
							}
						case *shapes.PolygonShape:
							collisionVector = GetRectanglePolygonCollision(eA, eB, *aS, *bS, ecsContainer)
							if !collisionVector.IsZero() {
								collisionFound = true
							}
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					case *shapes.CircleShape:
						switch bS := bColShape.(type) {
						case *shapes.RectangleShape:
							collisionVector = GetRectangleCircleCollision(eB, eA, *bS, *aS, ecsContainer)
							collisionVector = collisionVector.Multiply(-1)
							if !collisionVector.IsZero() {
								collisionFound = true
							}
						case *shapes.CircleShape:
							collisionVector = GetCircleCircleCollision(eA, eB, *aS, *bS, ecsContainer)
							if !collisionVector.IsZero() {
								collisionFound = true
							}
						case *shapes.PolygonShape:
							collisionVector = GetCirclePolygonCollision(eA, eB, *aS, *bS, ecsContainer)
							if !collisionVector.IsZero() {
								collisionFound = true
							}
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					case *shapes.PolygonShape:
						switch bS := bColShape.(type) {
						case *shapes.RectangleShape:
							collisionVector = GetRectanglePolygonCollision(eB, eA, *bS, *aS, ecsContainer)
							collisionVector = collisionVector.Multiply(-1)
							if !collisionVector.IsZero() {
								collisionFound = true
							}
						case *shapes.CircleShape:
							collisionVector = GetCirclePolygonCollision(eB, eA, *bS, *aS, ecsContainer)
							collisionVector = collisionVector.Multiply(-1)
							if !collisionVector.IsZero() {
								collisionFound = true
							}
						case *shapes.PolygonShape:
							collisionVector = GetPolygonPolygonCollision(eA, eB, *aS, *bS, ecsContainer)
							if !collisionVector.IsZero() {
								collisionFound = true
							}
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					}
				}
			}

			if !collisionVector.IsZero() {
				if _, ok := collisions[eA]; !ok {
					collisions[eA] = make(map[common.EntityId]common.Collision)
				}
				collisions[eA][eB] = common.Collision{
					Vector:    collisionVector,
					AShapeIdx: aCollidedIdx,
					BShapeIdx: bCollidedIdx,
				}
			}
		}
	}

	return collisions, nil
}

func GetMirrorAABBCollisions(
	aColManager ecs.IColliderManager,
	bColManager ecs.IColliderManager,
	proximateEntities map[common.EntityId][]common.EntityId,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId][]common.EntityId, error) {
	allCollisions := make(map[common.EntityId][]common.EntityId)

	aCollisions, err := GetAABBCollisions(aColManager, bColManager, proximateEntities, ecsContainer)
	if err != nil {
		return nil, err
	}

	bCollisions, err := GetAABBCollisions(bColManager, aColManager, proximateEntities, ecsContainer)
	if err != nil {
		return nil, err
	}

	for eA, collidedEntities := range aCollisions {
		if _, ok := allCollisions[eA]; !ok {
			allCollisions[eA] = []common.EntityId{}
		}
		allCollisions[eA] = append(allCollisions[eA], collidedEntities...)
	}

	for eB, collidedEntities := range bCollisions {
		if _, ok := allCollisions[eB]; !ok {
			allCollisions[eB] = []common.EntityId{}
		}
		allCollisions[eB] = append(allCollisions[eB], collidedEntities...)
	}

	return allCollisions, nil
}

func GetAABBCollisions(
	aColManager ecs.IColliderManager,
	bColManager ecs.IColliderManager,
	proximateEntities map[common.EntityId][]common.EntityId,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId][]common.EntityId, error) {
	collisions := make(map[common.EntityId][]common.EntityId)

	for e1, colEntities := range proximateEntities {
		for _, e2 := range colEntities {
			// TODO: Extract common code with GetCollisions()
			if e1 == e2 {
				continue
			}

			e1HasA := aColManager.HasCollider(e1, ecsContainer)
			e1HasB := bColManager.HasCollider(e1, ecsContainer)
			e2HasA := aColManager.HasCollider(e2, ecsContainer)
			e2HasB := bColManager.HasCollider(e2, ecsContainer)

			eA := common.EntityId(-1)
			eB := common.EntityId(-1)

			if e1HasA && e2HasB {
				eA = e1
				eB = e2
			} else if e1HasB && e2HasA {
				eA = e2
				eB = e1
			}

			if eA == -1 || eB == -1 {
				continue
			}

			aEnabled, err := aColManager.IsEnabled(eA, ecsContainer)
			if err != nil {
				log.Printf("Error checking if collider is enabled for entity %d: %v\n", eA, err)
				continue
			}

			if !aEnabled {
				continue
			}

			bEnabled, err := bColManager.IsEnabled(eB, ecsContainer)
			if err != nil {
				log.Printf("Error checking if collider is enabled for entity %d: %v\n", eB, err)
				continue
			}

			if !bEnabled {
				continue
			}

			aLayer, err := aColManager.GetLayer(eA, ecsContainer)
			if err != nil {
				log.Printf("Error getting collider layer for entity %d: %v\n", eA, err)
				continue
			}

			aMask, err := aColManager.GetMask(eA, ecsContainer)
			if err != nil {
				log.Printf("Error getting collider mask for entity %d: %v\n", eA, err)
				continue
			}

			aAABB, err := aColManager.GetWorldPaddedAABB(eA, ecsContainer)
			if err != nil {
				log.Printf("Error getting AABB for entity %d: %v\n", eA, err)
				continue
			}

			bLayer, err := bColManager.GetLayer(eB, ecsContainer)
			if err != nil {
				log.Printf("Error getting collider layer for entity %d: %v\n", eB, err)
				continue
			}

			bMask, err := bColManager.GetMask(eB, ecsContainer)
			if err != nil {
				log.Printf("Error getting collider mask for entity %d: %v\n", eB, err)
				continue
			}

			if aLayer&bMask == 0 || bLayer&aMask == 0 {
				continue
			}

			if collidedEntities, ok := collisions[eB]; ok {
				if slices.Contains(collidedEntities, eA) {
					continue
				}
			}

			bAABB, err := bColManager.GetWorldPaddedAABB(eB, ecsContainer)
			if err != nil {
				log.Printf("Error getting AABB for entity %d: %v\n", eB, err)
				continue
			}

			if utils.DetectAABBCollision(aAABB, bAABB) {
				v, ok := collisions[eA]
				if !ok {
					collisions[eA] = []common.EntityId{eB}
				}
				collisions[eA] = append(v, eB)
			}
		}
	}

	return collisions, nil
}
