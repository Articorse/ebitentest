package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"log"
)

// TODO: Add collision masks and check for valid collisions only in GetCollisions()
func ResolvePhysicsCollisions(
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
