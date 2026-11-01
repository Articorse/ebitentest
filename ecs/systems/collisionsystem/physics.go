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
				aTra, aTraOk := ecsContainer.Transforms.Get(eA)
				bTra, bTraOk := ecsContainer.Transforms.Get(eB)
				aVel, aVelOk := ecsContainer.Velocities.Get(eA)
				bVel, bVelOk := ecsContainer.Velocities.Get(eB)
				if !aTraOk || !bTraOk || !aVelOk || !bVelOk {
					log.Printf("Error resolving collision of mobs %d and %d: both need a transform and a velocity\n", eA, eB)
					continue
				}

				aTra.AddLocalPos(c.Vector.Multiply(-0.5))
				bTra.AddLocalPos(c.Vector.Multiply(0.5))

				normal := c.Vector.Normalized()
				relativeVelocity := aVel.LocalVector().Subtract(bVel.LocalVector())
				velocityAlongNormal := relativeVelocity.Dot(normal)

				if velocityAlongNormal < 0 {
					restitution := data.Bounciness
					impulseMagnitude := -(1 + restitution) * velocityAlongNormal
					impulse := normal.Multiply(impulseMagnitude)
					aVel.AddForce(impulse.Multiply(-0.5))
					bVel.AddForce(impulse.Multiply(0.5))
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

				mobTra, traOk := ecsContainer.Transforms.Get(mobE)
				mobVel, velOk := ecsContainer.Velocities.Get(mobE)
				if !traOk || !velOk {
					log.Printf("Error resolving collision of mob %d: it needs a transform and a velocity\n", mobE)
					continue
				}
				mobLocalVelVec := mobVel.LocalVector()

				var staLocalVelVec utils.Vec2f
				if staVel, ok := ecsContainer.Velocities.Get(staE); ok {
					staLocalVelVec = staVel.LocalVector()
				}

				mobTra.AddLocalPos(c.Vector)

				normal := c.Vector.Normalized()
				relativeVelocity := mobLocalVelVec.Subtract(staLocalVelVec)

				impulse := utils.CalculateImpulse(relativeVelocity, normal, data.Bounciness)
				if impulse.Length() > 0 {
					mobVel.SetLocalVector(mobLocalVelVec.Add(impulse))

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
