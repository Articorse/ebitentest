package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/components/hitboxes"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"log"
	"slices"
)

// TODO: Add collision masks and check for valid collisions only in GetCollisions()
func ResolveCollisions(
	collisions map[ecscommon.EntityId]map[ecscommon.EntityId]utils.Vec2,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
	velocities map[ecscommon.EntityId]*components.Velocity,
) (collisionsResolved uint64, err error) {
	tm := components.TransformManager{}
	vm := components.VelocityManager{}
	cm := components.ColliderManager{}

	for eA, cols := range collisions {
		for eB, colVector := range cols {
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
				colVector = colVector.Multiply(-1)
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

			tm.SetLocalPos(mobEnt, mobLocalPos.Add(colVector), transforms)

			normal := colVector.Normalized()
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
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) (map[ecscommon.EntityId]map[ecscommon.EntityId]utils.Vec2, error) {
	collisions := make(map[ecscommon.EntityId]map[ecscommon.EntityId]utils.Vec2)
	tm := components.TransformManager{}
	cm := components.ColliderManager{}

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

			aColHitboxes, err := cm.GetHitboxes(eA, colliders)
			if err != nil {
				log.Printf("Error getting collider hitboxes for entity %d: %v\n", eA, err)
				continue
			}

			bColHitboxes, err := cm.GetHitboxes(eB, colliders)
			if err != nil {
				log.Printf("Error getting collider hitboxes for entity %d: %v\n", eB, err)
				continue
			}

			var collisionVector utils.Vec2
			var aCollidedHitbox hitboxes.Hitbox
			var bCollidedHitbox hitboxes.Hitbox

			for _, aHitbox := range aColHitboxes {
				for _, bHitbox := range bColHitboxes {
					switch aH := aHitbox.(type) {
					case *hitboxes.RectangleHitbox:
						switch bH := bHitbox.(type) {
						case *hitboxes.RectangleHitbox:
							collisionVector = getRectangleRectangleCollision(eA, eB, *aH, *bH, transforms, parents)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						case *hitboxes.CircleHitbox:
							collisionVector = getRectangleCircleCollision(eA, eB, *aH, *bH, transforms, parents)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						case *hitboxes.PolygonHitbox:
							collisionVector = getRectanglePolygonCollision(eA, eB, *aH, *bH, transforms, parents)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						default:
							log.Printf("unsupported hitbox type for collision detection: %T", bH)
						}
					case *hitboxes.CircleHitbox:
						switch bH := bHitbox.(type) {
						case *hitboxes.RectangleHitbox:
							collisionVector = getRectangleCircleCollision(eB, eA, *bH, *aH, transforms, parents)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
							collisionVector = collisionVector.Multiply(-1)
						case *hitboxes.CircleHitbox:
							collisionVector = getCircleCircleCollision(eA, eB, *aH, *bH, transforms, parents)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						case *hitboxes.PolygonHitbox:
							collisionVector = getCirclePolygonCollision(eA, eB, *aH, *bH, transforms, parents)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						default:
							log.Printf("unsupported hitbox type for collision detection: %T", bH)
						}
					case *hitboxes.PolygonHitbox:
						switch bH := bHitbox.(type) {
						case *hitboxes.RectangleHitbox:
							collisionVector = getRectanglePolygonCollision(eB, eA, *bH, *aH, transforms, parents)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
							collisionVector = collisionVector.Multiply(-1)
						case *hitboxes.CircleHitbox:
							collisionVector = getCirclePolygonCollision(eB, eA, *bH, *aH, transforms, parents)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
							collisionVector = collisionVector.Multiply(-1)
						case *hitboxes.PolygonHitbox:
							collisionVector = getPolygonPolygonCollision(eA, eB, *aH, *bH, transforms, parents)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						default:
							log.Printf("unsupported hitbox type for collision detection: %T", bH)
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

			prevRelativePosVector := aWorldPrevPos.Add(aCollidedHitbox.GetOffset()).Subtract(bWorldPrevPos.Add(bCollidedHitbox.GetOffset()))
			if prevRelativePosVector.Dot(collisionVector) < 0 {
				collisionVector = collisionVector.Multiply(-1)
			}

			if !collisionVector.IsZero() {
				if _, ok := collisions[eA]; !ok {
					collisions[eA] = make(map[ecscommon.EntityId]utils.Vec2)
				}
				collisions[eA][eB] = collisionVector
			}
		}
	}

	return collisions, nil
}

func GetAABBCollisions(
	proximateEntities map[ecscommon.EntityId][]ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) (map[ecscommon.EntityId][]ecscommon.EntityId, error) {
	collisions := make(map[ecscommon.EntityId][]ecscommon.EntityId)
	tm := components.TransformManager{}
	cm := components.ColliderManager{}

	for eA, colEntities := range proximateEntities {
		for _, eB := range colEntities {
			if eA == eB {
				continue
			}

			if collidedEntities, ok := collisions[eB]; ok {
				if slices.Contains(collidedEntities, eA) {
					continue
				}
			}

			aWorldPos, err := tm.GetWorldPos(eA, transforms, parents)
			if err != nil {
				log.Printf("Error getting world position for entity %d: %v\n", eA, err)
				continue
			}

			bWorldPos, err := tm.GetWorldPos(eB, transforms, parents)
			if err != nil {
				log.Printf("Error getting world position for entity %d: %v\n", eB, err)
				continue
			}

			aAABB, err := cm.GetAABB(eA, colliders)
			if err != nil {
				log.Printf("Error getting AABB for entity %d: %v\n", eA, err)
				continue
			}

			bAABB, err := cm.GetAABB(eB, colliders)
			if err != nil {
				log.Printf("Error getting AABB for entity %d: %v\n", eB, err)
				continue
			}

			a := [2]utils.Vec2{
				utils.Vec2{X: aWorldPos.X + aAABB[0].X, Y: aWorldPos.Y + aAABB[0].Y},
				utils.Vec2{X: aWorldPos.X + aAABB[1].X, Y: aWorldPos.Y + aAABB[1].Y},
			}
			b := [2]utils.Vec2{
				utils.Vec2{X: bWorldPos.X + bAABB[0].X, Y: bWorldPos.Y + bAABB[0].Y},
				utils.Vec2{X: bWorldPos.X + bAABB[1].X, Y: bWorldPos.Y + bAABB[1].Y},
			}

			if detectAABBCollision(a, b) {
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

func detectAABBCollision(a, b [2]utils.Vec2) bool {
	minAx := a[0].X
	minAy := a[0].Y
	maxAx := a[0].X
	maxAy := a[0].Y
	for _, v := range a {
		if v.X < minAx {
			minAx = v.X
		}
		if v.X > maxAx {
			maxAx = v.X
		}
		if v.Y < minAy {
			minAy = v.Y
		}
		if v.Y > maxAy {
			maxAy = v.Y
		}
	}

	minBx := b[0].X
	minBy := b[0].Y
	maxBx := b[0].X
	maxBy := b[0].Y
	for _, v := range b {
		if v.X < minBx {
			minBx = v.X
		}
		if v.X > maxBx {
			maxBx = v.X
		}
		if v.Y < minBy {
			minBy = v.Y
		}
		if v.Y > maxBy {
			maxBy = v.Y
		}
	}

	return minAx <= maxBx && maxAx >= minBx && minAy <= maxBy && maxAy >= minBy
}
