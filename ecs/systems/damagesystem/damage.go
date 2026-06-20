package damagesystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/timerfuncs"
	"ebittest/utils"
	"fmt"
	"image/color"
	"log"
)

func Tick(ecs *ecs.ECS) {
	hpm := ecs.HitpointsManager

	for _, e := range ecs.Hitpoints.GetEntities() {
		invulCur, err := hpm.GetInvulCurrent(e, ecs)
		if err != nil {
			log.Printf("Error getting invulnerability current for entity %d: %v\n", e, err)
			continue
		}

		if invulCur == 0 {
			continue
		}

		err = hpm.TickInvul(e, ecs)
		if err != nil {
			log.Printf("Error ticking invulnerability for entity %d: %v\n", e, err)
			continue
		}
	}
}

func DealContactDamage(
	collisions map[common.EntityId]map[common.EntityId]common.Collision,
	ecs *ecs.ECS,
) (entitiesKilled uint64, err error) {
	vm := ecs.VelocityManager
	cdm := ecs.ContactDamageManager
	hpm := ecs.HitpointsManager

	for dmgE, cols := range collisions {
		disableColliderAfter := []common.EntityId{}

		for hitE, c := range cols {
			isInvul, err := hpm.IsInvul(hitE, ecs)
			if err != nil {
				log.Printf("Error checking invulnerability for entity %d: %v\n", hitE, err)
				continue
			}

			if isInvul {
				continue
			}

			damageTiers, err := cdm.GetDamageTiers(dmgE, ecs)
			if err != nil {
				log.Printf("Error getting damage tiers for entity %d: %v\n", dmgE, err)
				continue
			}

			if hitE == dmgE {
				continue
			}

			knockback, err := cdm.GetKnockback(dmgE, ecs)
			if err != nil {
				log.Printf("Error getting knockback for entity %d: %v\n", dmgE, err)
				continue
			}

			var dmgVelVector utils.Vec2
			if ecs.Velocities.HasComponent(dmgE) {
				dmgVelVector, err = vm.GetWorldVector(dmgE, ecs)
				if err != nil {
					log.Printf("Error getting ecs velocity vector for entity %d: %v\n", dmgE, err)
					continue
				}
			}

			dmgEForceNorm := dmgVelVector.Normalized()
			colForceNorm := c.Vector.Normalized()
			finalForceNorm := dmgEForceNorm.Multiply(0.5).Add(colForceNorm.Multiply(0.5))

			err = vm.AddForce(hitE, finalForceNorm.Multiply(knockback), ecs)
			if err != nil {
				log.Printf("Error applying knockback to entity %d: %v\n", hitE, err)
				continue
			}

			// TODO: Implement source
			// source, err := cdm.GetSource(dmgE, ecs.ContactDamages)
			// if err != nil {
			// 	log.Printf("Error getting source for entity %d: %v\n", dmgE, err)
			// 	continue
			// }

			shapeIdx := c.AShapeIdx

			if len(damageTiers) <= shapeIdx {
				log.Printf("Error: collider shape index %d out of range for damage tiers of entity %d\n", shapeIdx, dmgE)
				continue
			}

			dead, err := hpm.TakeDamage(hitE, damageTiers[shapeIdx], ecs)
			if err != nil {
				log.Printf("Error applying damage to entity %d: %v\n", hitE, err)
				continue
			}

			hitWorldPos, err := ecs.TransformManager.GetWorldPos(hitE, ecs)
			if err != nil {
				log.Printf("Error getting ecs position for entity %d: %v\n", hitE, err)
				continue
			}

			ftTraComp := ecs.NewTransformComponent(hitWorldPos, 1, 0)
			ftFtComp := ecs.NewFloatingTextComponent(fmt.Sprintf("%d", damageTiers[shapeIdx]), utils.Vec2{}, 12, color.RGBA{R: 255, G: 0, B: 255, A: 255})
			ftVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2{X: 0, Y: -1}, 1, 1)
			ftTimerComp, err := ecs.NewTimerComponent(1000, 1, timerfuncs.Selfdestruct)
			if err != nil {
				log.Fatal("error creating floating text timer component: ", err)
			}
			_ = ecs.AddEntity(ftTraComp, ftFtComp, ftVelComp, ftTimerComp)

			dieOnContact, err := cdm.GetDieOnContact(dmgE, ecs)
			if err != nil {
				log.Printf("Error getting die on contact for entity %d: %v\n", dmgE, err)
				continue
			}

			if dieOnContact {
				ecs.ScheduleRemoveEntity(dmgE)
			}

			singleTick, err := cdm.GetSingleTick(dmgE, ecs)
			if err != nil {
				log.Printf("Error getting single tick for entity %d: %v\n", dmgE, err)
				continue
			}

			if singleTick {
				disableColliderAfter = append(disableColliderAfter, hitE)
			}

			if dead {
				entitiesKilled++
				ecs.ScheduleRemoveEntity(hitE)
			}
		}

		for _, e := range disableColliderAfter {
			err := ecs.HurtboxColliderManager.SetEnabled(e, false, ecs)
			if err != nil {
				log.Printf("Error disabling hurtbox collider for entity %d: %v\n", e, err)
				continue
			}
		}
	}

	return entitiesKilled, nil
}
