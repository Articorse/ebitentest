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

type TileDamager interface {
	// False if the tile can't be damaged
	DamageTile(t utils.Vec2i, damage int) bool
}

// Single-tick hurtboxes are switched off at the end, once they've damaged everything they overlap, entities and tiles
// alike
func DealContactDamage(
	collisions []common.Collision,
	tileCollisions []common.TileCollision,
	tiles TileDamager,
	ecsCont *ecs.ECSContainer,
) (entitiesKilled uint64, err error) {
	for _, c := range collisions {
		dmgE, hitE := c.A, c.B
		cd, ok := ecsCont.ContactDamages.Get(dmgE)
		if !ok {
			log.Printf("Error dealing contact damage of entity %d: it has no contact damage\n", dmgE)
			continue
		}

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
		spawnDamageNumber(hitWorldPos, damageTiers[shapeIdx], ecsCont)

		if cd.DieOnContact() {
			ecsCont.ScheduleKillEntity(dmgE)
		}

		if dead {
			entitiesKilled++
			ecsCont.ScheduleKillEntity(hitE)
		}
	}

	dealTileDamage(tileCollisions, tiles, ecsCont)

	for e, cd := range ecs.Query1(&ecsCont.ContactDamages) {
		if !cd.SingleTick() {
			continue
		}
		if hurtbox, ok := ecsCont.HurtboxColliders.Get(e); ok {
			hurtbox.SetEnabled(false)
		}
	}

	return entitiesKilled, nil
}

func dealTileDamage(collisions []common.TileCollision, tiles TileDamager, ecsCont *ecs.ECSContainer) {
	for _, c := range collisions {
		cd, ok := ecsCont.ContactDamages.Get(c.E)
		if !ok {
			continue
		}

		if c.Impassable && cd.DieOnContact() {
			ecsCont.ScheduleKillEntity(c.E)
		}

		damageTiers := cd.DamageTiers()
		if len(damageTiers) <= c.ShapeIdx {
			log.Printf("Error: collider shape index %d out of range for damage tiers of entity %d\n", c.ShapeIdx, c.E)
			continue
		}

		if tiles.DamageTile(c.Tile, damageTiers[c.ShapeIdx]) {
			spawnDamageNumber(utils.TilePosToWorldPos(c.Tile), damageTiers[c.ShapeIdx], ecsCont)
		}
	}
}

func spawnDamageNumber(pos utils.Vec2f, damage int, ecsCont *ecs.ECSContainer) {
	ftTraComp := ecs.NewTransformComponent(pos, 1, 0)
	ftFtComp := ecs.NewFloatingTextComponent(fmt.Sprintf("%d", damage), utils.Vec2f{}, 12, color.RGBA{R: 255, G: 0, B: 255, A: 255})
	ftVelComp := ecs.NewVelocityComponentWithParams(utils.Vec2f{X: 0, Y: -1}, 1, 1)
	ftTimerComp, err := ecs.NewTimerComponent(1000, 1, timerfuncs.TimerFunc_Selfdestruct)
	if err != nil {
		log.Fatal("error creating floating text timer component: ", err)
	}
	if _, err := ecsCont.AddNewEntity(ftTraComp, ftFtComp, ftVelComp, ftTimerComp); err != nil {
		log.Printf("Error spawning damage number at %v: %v\n", pos, err)
	}
}
