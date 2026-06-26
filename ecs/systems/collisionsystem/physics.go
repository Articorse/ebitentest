package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

func ResolvePhysicsCollisions(
	collisions map[common.EntityId]map[common.EntityId]common.Collision,
	ecsContainer *ecs.ECSContainer,
) (collisionsResolved uint64, err error) {
	tm := ecsContainer.TransformManager
	vm := ecsContainer.VelocityManager
	cm := ecsContainer.PhysicsColliderManager

	for eA, cols := range collisions {
		for eB, c := range cols {
			aColType, err := cm.GetColliderType(eA, ecsContainer)
			if err != nil {
				log.Printf("Error getting collider type for entity %d: %v\n", eA, err)
				continue
			}

			bColType, err := cm.GetColliderType(eB, ecsContainer)
			if err != nil {
				log.Printf("Error getting collider type for entity %d: %v\n", eB, err)
				continue
			}

			if aColType == ecs.Collider_Trigger || bColType == ecs.Collider_Trigger {
				continue
			}

			// Mob-Mob
			if aColType == ecs.Collider_Mob && bColType == ecs.Collider_Mob {
				aLocalVelVec, err := vm.GetLocalVector(eA, ecsContainer)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eA, err)
					continue
				}
				bLocalVelVec, err := vm.GetLocalVector(eB, ecsContainer)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eB, err)
					continue
				}

				err = tm.AddLocalPos(eA, c.Vector.Multiply(-0.5), ecsContainer)
				if err != nil {
					log.Printf("Error adding local position for entity %d: %v\n", eA, err)
					continue
				}
				err = tm.AddLocalPos(eB, c.Vector.Multiply(0.5), ecsContainer)
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
					err = vm.AddForce(eA, impulse.Multiply(-0.5), ecsContainer)
					if err != nil {
						log.Printf("Error adding force to entity %d: %v\n", eA, err)
						continue
					}
					err = vm.AddForce(eB, impulse.Multiply(0.5), ecsContainer)
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

				mobLocalPos, err := tm.GetLocalPos(mobE, ecsContainer)
				if err != nil {
					log.Printf("Error getting local position for entity %d: %v\n", mobE, err)
					continue
				}
				mobLocalVelVec, err := vm.GetLocalVector(mobE, ecsContainer)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", mobE, err)
					continue
				}

				var staLocalVelVec utils.Vec2
				if ecsContainer.Velocities.HasComponent(staE) {
					staLocalVelVec, err = vm.GetLocalVector(staE, ecsContainer)
					if err != nil {
						log.Printf("Error getting local velocity vector for entity %d: %v\n", staE, err)
						continue
					}
				}

				err = tm.SetLocalPos(mobE, mobLocalPos.Add(c.Vector), ecsContainer)
				if err != nil {
					log.Printf("Error setting local position for entity %d: %v\n", mobE, err)
					continue
				}

				normal := c.Vector.Normalized()
				relativeVelocity := mobLocalVelVec.Subtract(staLocalVelVec)

				impulse := utils.CalculateImpulse(relativeVelocity, normal, data.Bounciness)
				if impulse.Length() > 0 {
					err = vm.SetLocalVector(mobE, mobLocalVelVec.Add(impulse), ecsContainer)
					if err != nil {
						log.Printf("Error setting local velocity vector for entity %d: %v\n", mobE, err)
						continue
					}

					collisionsResolved++
				}
			}

			// Static-Static
			if aColType == ecs.Collider_Static && bColType == ecs.Collider_Static {
				continue
			}
		}
	}

	return collisionsResolved, nil
}
