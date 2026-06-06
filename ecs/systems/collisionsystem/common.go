package collisionsystem

import (
	"ebittest/ecs/components"
	"ebittest/ecs/components/collidershapes"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"log"
	"slices"
)

func GetCollisions[T components.BaseColliderGetter](
	potentialCollisions map[ecscommon.EntityId][]ecscommon.EntityId,
	colliders map[ecscommon.EntityId]T,
	transforms map[ecscommon.EntityId]*components.Transform,
	velocities map[ecscommon.EntityId]*components.Velocity,
	parents map[ecscommon.EntityId]*components.Parent,
) (map[ecscommon.EntityId]map[ecscommon.EntityId]ecscommon.Collision, error) {
	collisions := make(map[ecscommon.EntityId]map[ecscommon.EntityId]ecscommon.Collision)
	tm := components.TransformManager{}
	cm := components.BaseColliderManager[T]{}

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

			aColShapes, err := cm.GetShapes(eA, colliders)
			if err != nil {
				log.Printf("Error getting collider shapes for entity %d: %v\n", eA, err)
				continue
			}

			bColShapes, err := cm.GetShapes(eB, colliders)
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
							collisionVector = getRectangleRectangleCollision(eA, eB, *aS, *bS, transforms, velocities, parents)
							collisionFound = true
						case *collidershapes.CircleShape:
							collisionVector = getRectangleCircleCollision(eA, eB, *aS, *bS, transforms, velocities, parents)
							collisionFound = true
						case *collidershapes.PolygonShape:
							collisionVector = getRectanglePolygonCollision(eA, eB, *aS, *bS, transforms, velocities, parents)
							collisionFound = true
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					case *collidershapes.CircleShape:
						switch bS := bColShape.(type) {
						case *collidershapes.RectangleShape:
							collisionVector = getRectangleCircleCollision(eB, eA, *bS, *aS, transforms, velocities, parents)
							collisionVector = collisionVector.Multiply(-1)
							collisionFound = true
						case *collidershapes.CircleShape:
							collisionVector = getCircleCircleCollision(eA, eB, *aS, *bS, transforms, velocities, parents)
							collisionFound = true
						case *collidershapes.PolygonShape:
							collisionVector = getCirclePolygonCollision(eA, eB, *aS, *bS, transforms, velocities, parents)
							collisionFound = true
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					case *collidershapes.PolygonShape:
						switch bS := bColShape.(type) {
						case *collidershapes.RectangleShape:
							collisionVector = getRectanglePolygonCollision(eB, eA, *bS, *aS, transforms, velocities, parents)
							collisionVector = collisionVector.Multiply(-1)
							collisionFound = true
						case *collidershapes.CircleShape:
							collisionVector = getCirclePolygonCollision(eB, eA, *bS, *aS, transforms, velocities, parents)
							collisionVector = collisionVector.Multiply(-1)
							collisionFound = true
						case *collidershapes.PolygonShape:
							collisionVector = getPolygonPolygonCollision(eA, eB, *aS, *bS, transforms, velocities, parents)
							collisionFound = true
						default:
							log.Printf("unsupported collider shape type for collision detection: %T", bS)
						}
					}
				}
			}

			aWorldPrevPos, err := tm.GetWorldPrevPos(eA, transforms, parents)
			if err != nil {
				log.Printf("Error getting world previous position for entity %d: %v\n", eA, err)
				continue
			}

			bWorldPrevPos, err := tm.GetWorldPrevPos(eB, transforms, parents)
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
					collisions[eA] = make(map[ecscommon.EntityId]ecscommon.Collision)
				}
				collisions[eA][eB] = ecscommon.Collision{
					Vector:    collisionVector,
					AShapeIdx: aCollidedIdx,
					BShapeIdx: bCollidedIdx,
				}
			}
		}
	}

	return collisions, nil
}

func GetAABBCollisions[T components.BaseColliderGetter](
	proximateEntities map[ecscommon.EntityId][]ecscommon.EntityId,
	colliders map[ecscommon.EntityId]T,
	collisionLayers map[ecscommon.EntityId]*components.CollisionLayer,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) (map[ecscommon.EntityId][]ecscommon.EntityId, error) {
	collisions := make(map[ecscommon.EntityId][]ecscommon.EntityId)
	cm := components.BaseColliderManager[T]{}
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

		aAABB, err := cm.GetWorldPaddedAABB(eA, colliders, transforms, parents)
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

			bAABB, err := cm.GetWorldPaddedAABB(eB, colliders, transforms, parents)
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
