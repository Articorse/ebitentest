package damagesystem

import (
	"ebittest/content/timerfuncs"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"image/color"
	"log"
)

func Tick(ecsCont *ecs.ECSContainer) {
	for _, hp := range ecs.Query1(&ecsCont.Hitpoints) {
		hp.TickInvul()
	}
}

func DealContactDamage(
	collisions map[common.EntityId]map[common.EntityId]common.Collision,
	ecsCont *ecs.ECSContainer,
) (entitiesKilled uint64, err error) {
	for dmgE, cols := range collisions {
		disableColliderAfter := []common.EntityId{}

		cd, ok := ecsCont.ContactDamages.Get(dmgE)
		if !ok {
			log.Printf("Error dealing contact damage of entity %d: it has no contact damage\n", dmgE)
			continue
		}

		for hitE, c := range cols {
			hp, ok := ecsCont.Hitpoints.Get(hitE)
			if !ok {
				log.Printf("Error dealing contact damage to entity %d: it has no hitpoints\n", hitE)
				continue
			}

			if hp.IsInvul() || hitE == dmgE {
				continue
			}

			damageTiers := cd.DamageTiers()
			knockback := cd.Knockback()

			var dmgVelVector utils.Vec2f
			if ecsCont.Velocities.HasComponent(dmgE) {
				dmgVelVector, err = ecsCont.WorldVector(dmgE)
				if err != nil {
					log.Printf("Error getting ecs velocity vector for entity %d: %v\n", dmgE, err)
					continue
				}
			}

			dmgEForceNorm := dmgVelVector.Normalized()
			colForceNorm := c.Vector.Normalized()
			finalForceNorm := dmgEForceNorm.Multiply(0.5).Add(colForceNorm.Multiply(0.5))

			hitVel, ok := ecsCont.Velocities.Get(hitE)
			if !ok {
				log.Printf("Error applying knockback to entity %d: it has no velocity\n", hitE)
				continue
			}
			hitVel.AddForce(finalForceNorm.Multiply(knockback))

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

			dead, err := ecsCont.Damage(hitE, damageTiers[shapeIdx])
			if err != nil {
				log.Printf("Error applying damage to entity %d: %v\n", hitE, err)
				continue
			}

			hitWorldPos, err := ecsCont.WorldPos(hitE)
			if err != nil {
				log.Printf("Error getting ecs position for entity %d: %v\n", hitE, err)
				continue
			}

			ftTraComp := ecs.NewTransformComponent(hitWorldPos, 1, 0)
			ftFtComp := ecs.NewFloatingTextComponent(fmt.Sprintf("%d", damageTiers[shapeIdx]), utils.Vec2f{}, 12, color.RGBA{R: 255, G: 0, B: 255, A: 255})
			ftVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2f{X: 0, Y: -1}, 1, 1)
			ftTimerComp, err := ecs.NewTimerComponent(1000, 1, timerfuncs.TimerFunc_Selfdestruct)
			if err != nil {
				log.Fatal("error creating floating text timer component: ", err)
			}
			if _, err := ecsCont.AddNewEntity(ftTraComp, ftFtComp, ftVelComp, ftTimerComp); err != nil {
				log.Printf("Error spawning floating text for entity %d: %v\n", hitE, err)
			}

			if cd.DieOnContact() {
				ecsCont.ScheduleKillEntity(dmgE)
			}

			if cd.SingleTick() {
				disableColliderAfter = append(disableColliderAfter, hitE)
			}

			if dead {
				entitiesKilled++
				ecsCont.ScheduleKillEntity(hitE)
			}
		}

		for _, e := range disableColliderAfter {
			hurtbox, ok := ecsCont.HurtboxColliders.Get(e)
			if !ok {
				log.Printf("Error disabling hurtbox collider for entity %d: it has none\n", e)
				continue
			}
			hurtbox.SetEnabled(false)
		}
	}

	return entitiesKilled, nil
}
