package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/components/collidershapes"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"log"
)

// TODO: Add collision masks and check for valid collisions only in GetCollisions()
func ResolveCollisions(
	collisions map[ecscommon.EntityId]map[ecscommon.EntityId]ecscommon.Collision,
	colliders map[ecscommon.EntityId]*components.PhysicsCollider,
	transforms map[ecscommon.EntityId]*components.Transform,
	velocities map[ecscommon.EntityId]*components.Velocity,
) (collisionsResolved uint64, err error) {
	tm := components.TransformManager{}
	vm := components.VelocityManager{}
	cm := components.PhysicsColliderManager{}

	for eA, cols := range collisions {
		for eB, c := range cols {
			aColType, err := cm.GetColliderType(eA, colliders)
			if err != nil {
				log.Printf("Error getting collider type for entity %d: %v\n", eA, err)
				continue
			}

			bColType, err := cm.GetColliderType(eB, colliders)
			if err != nil {
				log.Printf("Error getting collider type for entity %d: %v\n", eB, err)
				continue
			}

			var mobEnt ecscommon.EntityId
			var mobLocalPos utils.Vec2
			var mobLocalVelVec utils.Vec2
			var staticLocalVelVec utils.Vec2

			if aColType == components.Collider_Mob && bColType == components.Collider_Static {
				mobEnt = eA
				mobLocalPos, err = tm.GetLocalPos(eA, transforms)
				if err != nil {
					log.Printf("Error getting local position for entity %d: %v\n", eA, err)
					continue
				}
				mobLocalVelVec, err = vm.GetLocalVector(eA, velocities)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eA, err)
					continue
				}
				staticLocalVelVec, err = vm.GetLocalVector(eB, velocities)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eB, err)
					continue
				}
			} else if bColType == components.Collider_Mob && aColType == components.Collider_Static {
				c.Vector = c.Vector.Multiply(-1)
				mobEnt = eB
				mobLocalPos, err = tm.GetLocalPos(eB, transforms)
				if err != nil {
					log.Printf("Error getting local position for entity %d: %v\n", eB, err)
					continue
				}
				mobLocalVelVec, err = vm.GetLocalVector(eB, velocities)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eB, err)
					continue
				}
				staticLocalVelVec, err = vm.GetLocalVector(eA, velocities)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eA, err)
					continue
				}
			} else {
				continue
			}

			tm.SetLocalPos(mobEnt, mobLocalPos.Add(c.Vector), transforms)

			normal := c.Vector.Normalized()
			relativeVelocity := mobLocalVelVec.Subtract(staticLocalVelVec)
			velocityAlongNormal := relativeVelocity.Dot(normal)

			if velocityAlongNormal < 0 {
				restitution := data.Bounciness
				impulseMagnitude := -(1 + restitution) * velocityAlongNormal
				impulse := normal.Multiply(impulseMagnitude)
				vm.SetLocalVector(mobEnt, mobLocalVelVec.Add(impulse), velocities)
			}

			collisionsResolved++
		}
	}

	return collisionsResolved, nil
}

func GetCollisions(
	potentialCollisions map[ecscommon.EntityId][]ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*components.PhysicsCollider,
	transforms map[ecscommon.EntityId]*components.Transform,
	velocities map[ecscommon.EntityId]*components.Velocity,
	parents map[ecscommon.EntityId]*components.Parent,
) (map[ecscommon.EntityId]map[ecscommon.EntityId]ecscommon.Collision, error) {
	collisions := make(map[ecscommon.EntityId]map[ecscommon.EntityId]ecscommon.Collision)
	tm := components.TransformManager{}
	cm := components.PhysicsColliderManager{}

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
					Vector:     collisionVector,
					AShapeIdx: aCollidedIdx,
					BShapeIdx: bCollidedIdx,
				}
			}
		}
	}

	return collisions, nil
}
