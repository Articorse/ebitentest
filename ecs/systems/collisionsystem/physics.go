package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

// TODO: Add collision masks and check for valid collisions only in GetCollisions()
func ResolvePhysicsCollisions(
	collisions map[common.EntityId]map[common.EntityId]common.Collision,
	world *ecs.World,
) (collisionsResolved uint64, err error) {
	tm := ecs.TransformManager{}
	vm := ecs.VelocityManager{}
	cm := ecs.PhysicsColliderManager{}

	for eA, cols := range collisions {
		for eB, c := range cols {
			aColType, err := cm.GetColliderType(eA, world.PhysicsColliders)
			if err != nil {
				log.Printf("Error getting collider type for entity %d: %v\n", eA, err)
				continue
			}

			bColType, err := cm.GetColliderType(eB, world.PhysicsColliders)
			if err != nil {
				log.Printf("Error getting collider type for entity %d: %v\n", eB, err)
				continue
			}

			var mobEnt common.EntityId
			var mobLocalPos utils.Vec2
			var mobLocalVelVec utils.Vec2
			var staticLocalVelVec utils.Vec2

			if aColType == ecs.Collider_Mob && bColType == ecs.Collider_Static {
				mobEnt = eA
				mobLocalPos, err = tm.GetLocalPos(eA, world.Transforms)
				if err != nil {
					log.Printf("Error getting local position for entity %d: %v\n", eA, err)
					continue
				}
				mobLocalVelVec, err = vm.GetLocalVector(eA, world.Velocities)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eA, err)
					continue
				}
				staticLocalVelVec, err = vm.GetLocalVector(eB, world.Velocities)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eB, err)
					continue
				}
			} else if bColType == ecs.Collider_Mob && aColType == ecs.Collider_Static {
				c.Vector = c.Vector.Multiply(-1)
				mobEnt = eB
				mobLocalPos, err = tm.GetLocalPos(eB, world.Transforms)
				if err != nil {
					log.Printf("Error getting local position for entity %d: %v\n", eB, err)
					continue
				}
				mobLocalVelVec, err = vm.GetLocalVector(eB, world.Velocities)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eB, err)
					continue
				}
				staticLocalVelVec, err = vm.GetLocalVector(eA, world.Velocities)
				if err != nil {
					log.Printf("Error getting local velocity vector for entity %d: %v\n", eA, err)
					continue
				}
			} else {
				continue
			}

			tm.SetLocalPos(mobEnt, mobLocalPos.Add(c.Vector), world.Transforms)

			normal := c.Vector.Normalized()
			relativeVelocity := mobLocalVelVec.Subtract(staticLocalVelVec)
			velocityAlongNormal := relativeVelocity.Dot(normal)

			if velocityAlongNormal < 0 {
				restitution := data.Bounciness
				impulseMagnitude := -(1 + restitution) * velocityAlongNormal
				impulse := normal.Multiply(impulseMagnitude)
				vm.SetLocalVector(mobEnt, mobLocalVelVec.Add(impulse), world.Velocities)
			}

			collisionsResolved++
		}
	}

	return collisionsResolved, nil
}
