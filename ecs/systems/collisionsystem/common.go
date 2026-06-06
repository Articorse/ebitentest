package collisionsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/collidershapes"
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
	tm := ecs.TransformManager{}

	for eA, colEntities := range potentialCollisions {
		for _, eB := range colEntities {
			if eA == eB {
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
			var aCollidedShapes collidershapes.Shape
			var bCollidedShapes collidershapes.Shape
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

					aCollidedShapes = aColShape
					bCollidedShapes = bColShape
					aCollidedIdx = aColShapeIdx
					bCollidedIdx = bColShapeIdx

					switch aS := aColShape.(type) {
					case *collidershapes.RectangleShape:
						switch bS := bColShape.(type) {
						case *collidershapes.RectangleShape:
							collisionVector = getRectangleRectangleCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						case *collidershapes.CircleShape:
							collisionVector = getRectangleCircleCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						case *collidershapes.PolygonShape:
							collisionVector = getRectanglePolygonCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					case *collidershapes.CircleShape:
						switch bS := bColShape.(type) {
						case *collidershapes.RectangleShape:
							collisionVector = getRectangleCircleCollision(eB, eA, *bS, *aS, world)
							collisionVector = collisionVector.Multiply(-1)
							collisionFound = true
						case *collidershapes.CircleShape:
							collisionVector = getCircleCircleCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						case *collidershapes.PolygonShape:
							collisionVector = getCirclePolygonCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					case *collidershapes.PolygonShape:
						switch bS := bColShape.(type) {
						case *collidershapes.RectangleShape:
							collisionVector = getRectanglePolygonCollision(eB, eA, *bS, *aS, world)
							collisionVector = collisionVector.Multiply(-1)
							collisionFound = true
						case *collidershapes.CircleShape:
							collisionVector = getCirclePolygonCollision(eB, eA, *bS, *aS, world)
							collisionVector = collisionVector.Multiply(-1)
							collisionFound = true
						case *collidershapes.PolygonShape:
							collisionVector = getPolygonPolygonCollision(eA, eB, *aS, *bS, world)
							collisionFound = true
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					}
				}
			}

			aWorldPrevPos, err := tm.GetWorldPrevPos(eA, world.Transforms, world.Parents)
			if err != nil {
				log.Printf("Error getting world previous position for entity %d: %v\n", eA, err)
				continue
			}

			bWorldPrevPos, err := tm.GetWorldPrevPos(eB, world.Transforms, world.Parents)
			if err != nil {
				log.Printf("Error getting world previous position for entity %d: %v\n", eB, err)
				continue
			}

			prevRelativePosVector := aWorldPrevPos.Add(aCollidedShapes.GetOffset()).Subtract(bWorldPrevPos.Add(bCollidedShapes.GetOffset()))
			if prevRelativePosVector.Dot(collisionVector) < 0 {
				collisionVector = collisionVector.Multiply(-1)
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

func GetAABBCollisions(
	aColManager ecs.IColliderManager,
	bColManager ecs.IColliderManager,
	proximateEntities map[common.EntityId][]common.EntityId,
	world *ecs.World,
) (map[common.EntityId][]common.EntityId, error) {
	collisions := make(map[common.EntityId][]common.EntityId)
	clm := ecs.CollisionLayersManager{}

	for eA, colEntities := range proximateEntities {
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

		for _, eB := range colEntities {
			if eA == eB {
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
