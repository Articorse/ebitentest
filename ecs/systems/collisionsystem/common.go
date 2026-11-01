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
	aColliders ecs.ColliderSet,
	bColliders ecs.ColliderSet,
	potentialCollisions map[common.EntityId][]common.EntityId,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId]map[common.EntityId]common.Collision, error) {
	collisions := make(map[common.EntityId]map[common.EntityId]common.Collision)
	for e1, colEntities := range potentialCollisions {
		for _, e2 := range colEntities {
			eA, eB, aCol, bCol, ok := orient(e1, e2, aColliders, bColliders)
			if !ok {
				continue
			}

			if collidedEntities, ok := collisions[eB]; ok {
				if _, ok := collidedEntities[eA]; ok {
					continue
				}
			}

			aColShapes := aCol.Shapes()
			bColShapes := bCol.Shapes()

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
	aColliders ecs.ColliderSet,
	bColliders ecs.ColliderSet,
	proximateEntities map[common.EntityId][]common.EntityId,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId][]common.EntityId, error) {
	allCollisions := make(map[common.EntityId][]common.EntityId)

	aCollisions, err := GetAABBCollisions(aColliders, bColliders, proximateEntities, ecsContainer)
	if err != nil {
		return nil, err
	}

	bCollisions, err := GetAABBCollisions(bColliders, aColliders, proximateEntities, ecsContainer)
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
	aColliders ecs.ColliderSet,
	bColliders ecs.ColliderSet,
	proximateEntities map[common.EntityId][]common.EntityId,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId][]common.EntityId, error) {
	collisions := make(map[common.EntityId][]common.EntityId)

	for e1, colEntities := range proximateEntities {
		for _, e2 := range colEntities {
			eA, eB, aCol, bCol, ok := orient(e1, e2, aColliders, bColliders)
			if !ok {
				continue
			}

			if !aCol.Enabled() || !bCol.Enabled() {
				continue
			}

			aAABB, err := ecsContainer.WorldAABB(eA, aCol.LocalPaddedAABB())
			if err != nil {
				log.Printf("Error getting AABB for entity %d: %v\n", eA, err)
				continue
			}

			if aCol.Layer()&bCol.Mask() == 0 || bCol.Layer()&aCol.Mask() == 0 {
				continue
			}

			if collidedEntities, ok := collisions[eB]; ok {
				if slices.Contains(collidedEntities, eA) {
					continue
				}
			}

			bAABB, err := ecsContainer.WorldAABB(eB, bCol.LocalPaddedAABB())
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

func orient(
	e1, e2 common.EntityId,
	aColliders, bColliders ecs.ColliderSet,
) (eA, eB common.EntityId, aCol, bCol ecs.Collider, ok bool) {
	if e1 == e2 {
		return -1, -1, nil, nil, false
	}

	if aCol, ok := aColliders.Collider(e1); ok {
		if bCol, ok := bColliders.Collider(e2); ok {
			return e1, e2, aCol, bCol, true
		}
	}

	if aCol, ok := aColliders.Collider(e2); ok {
		if bCol, ok := bColliders.Collider(e1); ok {
			return e2, e1, aCol, bCol, true
		}
	}

	return -1, -1, nil, nil, false
}
