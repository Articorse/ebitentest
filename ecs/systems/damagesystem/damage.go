package damagesystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"log"
)

func DealContactDamage(
	collisions map[common.EntityId]map[common.EntityId]common.Collision,
	world *ecs.World,
) (entitiesKilled uint64, err error) {
	vm := ecs.VelocityManager{}
	cdm := ecs.ContactDamageManager{}
	hpm := ecs.HitpointsManager{}

	for eA, cols := range collisions {
		for eB, c := range cols {
			var hitE common.EntityId
			var dmgE common.EntityId

			hitE = eA
			_, err := hpm.GetCurrent(eA, world.Hitpoints)
			if err != nil {
				_, err = hpm.GetCurrent(eB, world.Hitpoints)
				if err != nil {
					hitE = eB
				} else {
					continue
				}
			}

			dmgE = eA
			damageTiers, _ := cdm.GetDamageTiers(eA, world.ContactDamages)
			if damageTiers == nil {
				damageTiers, _ = cdm.GetDamageTiers(eB, world.ContactDamages)
				dmgE = eB
			}
			if damageTiers == nil {
				continue
			}

			knockback, err := cdm.GetKnockback(dmgE, world.ContactDamages)
			if err != nil {
				log.Printf("Error getting knockback for entity %d: %v\n", dmgE, err)
				continue
			}

			// TODO: Implement source
			// source, err := cdm.GetSource(dmgE, world.ContactDamages)
			// if err != nil {
			// 	log.Printf("Error getting source for entity %d: %v\n", dmgE, err)
			// 	continue
			// }

			var shapeIdx int
			switch dmgE {
			case eA:
				shapeIdx = c.AShapeIdx
			case eB:
				shapeIdx = c.BShapeIdx
			default:
				log.Printf("Error determining collider shape index for entities %d and %d\n", eA, eB)
				continue
			}

			if len(damageTiers) <= shapeIdx {
				log.Printf("Error: collider shape index %d out of range for damage tiers of entity %d\n", shapeIdx, dmgE)
				continue
			}

			dead, err := hpm.TakeDamage(hitE, damageTiers[shapeIdx], world.Hitpoints)
			if err != nil {
				log.Printf("Error applying damage to entity %d: %v\n", hitE, err)
				continue
			}

			err = world.RemoveEntity(dmgE)
			if err != nil {
				log.Printf("Error removing entity %d: %v\n", dmgE, err)
			}

			if dead {
				entitiesKilled++
				err = world.RemoveEntity(hitE)
				if err != nil {
					log.Printf("Error removing entity %d: %v\n", hitE, err)
				}
			}

			err = vm.AddForce(hitE, c.Vector.Normalized().Multiply(knockback), world.Velocities)
			if err != nil {
				log.Printf("Error applying knockback to entity %d: %v\n", hitE, err)
				continue
			}
		}
	}

	return entitiesKilled, nil
}
