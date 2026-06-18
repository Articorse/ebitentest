package damagesystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

func Tick(world *ecs.World) {
	hpm := world.HitpointsManager

	for _, e := range world.Hitpoints.GetEntities() {
		invulCur, err := hpm.GetInvulCurrent(e, world)
		if err != nil {
			log.Printf("Error getting invulnerability current for entity %d: %v\n", e, err)
			continue
		}

		if invulCur == 0 {
			continue
		}

		err = hpm.TickInvul(e, world)
		if err != nil {
			log.Printf("Error ticking invulnerability for entity %d: %v\n", e, err)
			continue
		}
	}
}

func DealContactDamage(
	collisions map[common.EntityId]map[common.EntityId]common.Collision,
	world *ecs.World,
) (entitiesKilled uint64, err error) {
	vm := world.VelocityManager
	cdm := world.ContactDamageManager
	hpm := world.HitpointsManager

	for dmgE, cols := range collisions {
		deleteAfter := []common.EntityId{}

		for hitE, c := range cols {
			isInvul, err := hpm.IsInvul(hitE, world)
			if err != nil {
				log.Printf("Error checking invulnerability for entity %d: %v\n", hitE, err)
				continue
			}

			if isInvul {
				continue
			}

			damageTiers, err := cdm.GetDamageTiers(dmgE, world)
			if err != nil {
				log.Printf("Error getting damage tiers for entity %d: %v\n", dmgE, err)
				continue
			}

			if hitE == dmgE {
				continue
			}

			knockback, err := cdm.GetKnockback(dmgE, world)
			if err != nil {
				log.Printf("Error getting knockback for entity %d: %v\n", dmgE, err)
				continue
			}

			var dmgVelVector utils.Vec2
			if world.Velocities.HasComponent(dmgE) {
				dmgVelVector, err = vm.GetWorldVector(dmgE, world)
				if err != nil {
					log.Printf("Error getting world velocity vector for entity %d: %v\n", dmgE, err)
					continue
				}
			}

			dmgEForceNorm := dmgVelVector.Normalized()
			colForceNorm := c.Vector.Normalized()
			finalForceNorm := dmgEForceNorm.Multiply(0.5).Add(colForceNorm.Multiply(0.5))

			err = vm.AddForce(hitE, finalForceNorm.Multiply(knockback), world)
			if err != nil {
				log.Printf("Error applying knockback to entity %d: %v\n", hitE, err)
				continue
			}

			// TODO: Implement source
			// source, err := cdm.GetSource(dmgE, world.ContactDamages)
			// if err != nil {
			// 	log.Printf("Error getting source for entity %d: %v\n", dmgE, err)
			// 	continue
			// }

			shapeIdx := c.BShapeIdx

			if len(damageTiers) <= shapeIdx {
				log.Printf("Error: collider shape index %d out of range for damage tiers of entity %d\n", shapeIdx, dmgE)
				continue
			}

			dead, err := hpm.TakeDamage(hitE, damageTiers[shapeIdx], world)
			if err != nil {
				log.Printf("Error applying damage to entity %d: %v\n", hitE, err)
				continue
			}

			dieOnContact, err := cdm.GetDieOnContact(dmgE, world)
			if err != nil {
				log.Printf("Error getting die on contact for entity %d: %v\n", dmgE, err)
				continue
			}

			if dieOnContact {
				world.ScheduleRemoveEntity(dmgE)
			}

			singleTick, err := cdm.GetSingleTick(dmgE, world)
			if err != nil {
				log.Printf("Error getting single tick for entity %d: %v\n", dmgE, err)
				continue
			}

			if singleTick {
				deleteAfter = append(deleteAfter, hitE)
			}

			if dead {
				entitiesKilled++
				world.ScheduleRemoveEntity(hitE)
			}
		}

		for _, e := range deleteAfter {
			world.HurtboxColliderManager.SetEnabled(e, false, world)
		}
	}

	return entitiesKilled, nil
}
