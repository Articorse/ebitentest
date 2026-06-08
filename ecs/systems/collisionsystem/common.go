package collisionsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/shapes"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
	"slices"
)

func GetCollisions(
	aColManager ecs.IColliderManager,
	bColManager ecs.IColliderManager,
	potentialCollisions map[common.EntityId][]common.EntityId,
	world *ecs.World,
) (map[common.EntityId]map[common.EntityId]common.Collision, error) {
	collisions := make(map[common.EntityId]map[common.EntityId]common.Collision)
	for e1, colEntities := range potentialCollisions {
		eA := common.EntityId(-1)
		eB := common.EntityId(-1)

		if aColManager.HasCollider(e1, world) {
			eA = e1
		} else if bColManager.HasCollider(e1, world) {
			eB = e1
		}

		for _, e2 := range colEntities {
			if e1 == e2 {
				continue
			}

			if bColManager.HasCollider(e2, world) {
				eB = e2
			} else if aColManager.HasCollider(e2, world) {
				eA = e2
			}

			if eA == -1 || eB == -1 {
				continue
			}

			if collidedEntities, ok := collisions[eB]; ok {
				if _, ok := collidedEntities[eA]; ok {
					continue
				}
			}

			aColShapes, err := aColManager.GetShapes(eA, world)
			if err != nil {
				log.Printf("Error getting collider shapes for entity %d: %v\n", eA, err)
				continue
			}

			bColShapes, err := bColManager.GetShapes(eB, world)
			if err != nil {
				log.Printf("Error getting collider shapes for entity %d: %v\n", eB, err)
				continue
			}

			collisionFound := false
			var collisionVector utils.Vec2
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
							collisionVector = getRectangleRectangleCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						case *shapes.CircleShape:
							collisionVector = getRectangleCircleCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						case *shapes.PolygonShape:
							collisionVector = getRectanglePolygonCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					case *shapes.CircleShape:
						switch bS := bColShape.(type) {
						case *shapes.RectangleShape:
							collisionVector = getRectangleCircleCollision(eB, eA, *bS, *aS, world)
							collisionVector = collisionVector.Multiply(-1)
							collisionFound = true
						case *shapes.CircleShape:
							collisionVector = getCircleCircleCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						case *shapes.PolygonShape:
							collisionVector = getCirclePolygonCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					case *shapes.PolygonShape:
						switch bS := bColShape.(type) {
						case *shapes.RectangleShape:
							collisionVector = getRectanglePolygonCollision(eB, eA, *bS, *aS, world)
							collisionVector = collisionVector.Multiply(-1)
							collisionFound = true
						case *shapes.CircleShape:
							collisionVector = getCirclePolygonCollision(eB, eA, *bS, *aS, world)
							collisionVector = collisionVector.Multiply(-1)
							collisionFound = true
						case *shapes.PolygonShape:
							collisionVector = getPolygonPolygonCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
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
	world *ecs.World,
) (map[common.EntityId][]common.EntityId, error) {
	allCollisions := make(map[common.EntityId][]common.EntityId)

	aCollisions, err := GetAABBCollisions(aColManager, bColManager, proximateEntities, world)
	if err != nil {
		return nil, err
	}

	bCollisions, err := GetAABBCollisions(bColManager, aColManager, proximateEntities, world)
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
	world *ecs.World,
) (map[common.EntityId][]common.EntityId, error) {
	collisions := make(map[common.EntityId][]common.EntityId)
	clm := ecs.CollisionLayersManager{}

	for e1, colEntities := range proximateEntities {
		eA := common.EntityId(-1)
		eB := common.EntityId(-1)

		if aColManager.HasCollider(e1, world) {
			eA = e1
		} else if bColManager.HasCollider(e1, world) {
			eB = e1
		}

		for _, e2 := range colEntities {
			if e1 == e2 {
				continue
			}

			if bColManager.HasCollider(e2, world) {
				eB = e2
			} else if aColManager.HasCollider(e2, world) {
				eA = e2
			}

			if eA == -1 || eB == -1 {
				continue
			}

			aLayers, err := clm.GetLayers(eA, world.CollisionLayers)
			if err != nil {
				log.Printf("Error getting collider layers for entity %d: %v\n", eA, err)
				continue
			}

			aMask, err := clm.GetMask(eA, world.CollisionLayers)
			if err != nil {
				log.Printf("Error getting collider mask for entity %d: %v\n", eA, err)
				continue
			}

			aAABB, err := aColManager.GetWorldPaddedAABB(eA, world)
			if err != nil {
				log.Printf("Error getting AABB for entity %d: %v\n", eA, err)
				continue
			}

			bLayers, err := clm.GetLayers(eB, world.CollisionLayers)
			if err != nil {
				log.Printf("Error getting collider layers for entity %d: %v\n", eB, err)
				continue
			}

			bMask, err := clm.GetMask(eB, world.CollisionLayers)
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

			bAABB, err := bColManager.GetWorldPaddedAABB(eB, world)
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
