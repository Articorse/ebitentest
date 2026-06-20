package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"log"
)

func ResolvePhysicsCollisions(
	collisions map[common.EntityId]map[common.EntityId]common.Collision,
	ecs *ecs.ECS,
) (collisionsResolved uint64, err error) {
	tm := ecs.TransformManager
	vm := ecs.VelocityManager
	cm := ecs.PhysicsColliderManager

	for eA, cols := range collisions {
		for eB, c := range cols {
			aColType, err := cm.GetColliderType(eA, ecs)
			if err != nil {
				log.Printf("Error getting collider type for entity %d: %v\n", eA, err)
				continue
			}

			bColType, err := cm.GetColliderType(eB, ecs)
			if err != nil {
				log.Printf("Error getting collider type for entity %d: %v\n", eB, err)
				continue
			}

			if aColType == ecs.Collider_Trigger || bColType == ecs.Collider_Trigger {
				continue
			}

			// Mob-Mob
			if aColType == ecs.Collider_Mob && bColType == ecs.Collider_Mob {
				aLocalVelVec, err := vm.GetLocalVector(eA, ecs)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eA, err)
					continue
				}
				bLocalVelVec, err := vm.GetLocalVector(eB, ecs)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eB, err)
					continue
				}

				err = tm.AddLocalPos(eA, c.Vector.Multiply(-0.5), ecs)
				if err != nil {
					log.Printf("Error adding local position for entity %d: %v\n", eA, err)
					continue
				}
				err = tm.AddLocalPos(eB, c.Vector.Multiply(0.5), ecs)
				if err != nil {
					log.Printf("Error adding local position for entity %d: %v\n", eB, err)
					continue
				}

				normal := c.Vector.Normalized()
				relativeVelocity := aLocalVelVec.Subtract(bLocalVelVec)
				velocityAlongNormal := relativeVelocity.Dot(normal)

				if velocityAlongNormal < 0 {
					restitution := data.Bounciness
					impulseMagnitude := -(1 + restitution) * velocityAlongNormal
					impulse := normal.Multiply(impulseMagnitude)
					err = vm.AddForce(eA, impulse.Multiply(-0.5), ecs)
					if err != nil {
						log.Printf("Error adding force to entity %d: %v\n", eA, err)
						continue
					}
					err = vm.AddForce(eB, impulse.Multiply(0.5), ecs)
					if err != nil {
						log.Printf("Error adding force to entity %d: %v\n", eB, err)
						continue
					}
				}

				collisionsResolved++
				continue
			}

			// Mob-Static
			if (aColType == ecs.Collider_Mob && bColType == ecs.Collider_Static) ||
				(aColType == ecs.Collider_Static && bColType == ecs.Collider_Mob) {
				eAIsMob := aColType == ecs.Collider_Mob

				var mobE common.EntityId
				var staE common.EntityId

				if eAIsMob {
					mobE = eA
					staE = eB
					c.Vector = c.Vector.Multiply(-1)
				} else {
					mobE = eB
					staE = eA
				}

				mobLocalPos, err := tm.GetLocalPos(mobE, ecs)
				if err != nil {
					log.Printf("Error getting local position for entity %d: %v\n", mobE, err)
					continue
				}
				mobLocalVelVec, err := vm.GetLocalVector(mobE, ecs)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", mobE, err)
					continue
				}
				staLocalVelVec, err := vm.GetLocalVector(staE, ecs)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", staE, err)
					continue
				}

				err = tm.SetLocalPos(mobE, mobLocalPos.Add(c.Vector), ecs)
				if err != nil {
					log.Printf("Error setting local position for entity %d: %v\n", mobE, err)
					continue
				}

				normal := c.Vector.Normalized()
				relativeVelocity := mobLocalVelVec.Subtract(staLocalVelVec)
				velocityAlongNormal := relativeVelocity.Dot(normal)

				if velocityAlongNormal < 0 {
					restitution := data.Bounciness
					impulseMagnitude := -(1 + restitution) * velocityAlongNormal
					impulse := normal.Multiply(impulseMagnitude)
					err = vm.SetLocalVector(mobE, mobLocalVelVec.Add(impulse), ecs)
					if err != nil {
						log.Printf("Error setting local velocity vector for entity %d: %v\n", mobE, err)
						continue
					}
				}

				collisionsResolved++
				continue
			}

			// Static-Static
			if aColType == ecs.Collider_Static && bColType == ecs.Collider_Static {
				continue
			}
		}
	}

	return collisionsResolved, nil
}
