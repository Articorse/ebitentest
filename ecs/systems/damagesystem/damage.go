package damagesystem

import (
	"ebittest/content/effects"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

func Tick(ecsCont *ecs.ECSContainer) {
	for _, hp := range ecs.Query1(&ecsCont.Hitpoints, ecs.Without(&ecsCont.Disabled)) {
		hp.TickInvul()
	}
}

func DealContactDamage(
	collisions []common.Collision,
	tileCollisions []common.TileCollision,
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

		if hp.IsInvul() || hitE == dmgE || !cd.CanHitEntity(hitE) {
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
		cd.RecordEntityHit(hitE)

		hitWorldPos, err := ecsCont.WorldPos(hitE)
		if err != nil {
			log.Printf("Error getting ecs position for entity %d: %v\n", hitE, err)
			continue
		}
		if err := effects.SpawnDamageNumber(hitWorldPos, damageTiers[shapeIdx], ecsCont); err != nil {
			log.Println(err)
		}

		if cd.DieOnContact() {
			ecsCont.ScheduleKillEntity(dmgE)
		}

		if dead {
			entitiesKilled++
			ecsCont.ScheduleKillEntity(hitE)
		}
	}

	dealTileDamage(tileCollisions, ecsCont)

	// Aged after dealing damage, so that a source deals damage in the tick it's created
	for _, cd := range ecs.Query1(&ecsCont.ContactDamages, ecs.Without(&ecsCont.Disabled)) {
		cd.TickAge()
	}

	return entitiesKilled, nil
}

func dealTileDamage(collisions []common.TileCollision, ecsCont *ecs.ECSContainer) {
	for _, c := range collisions {
		cd, ok := ecsCont.ContactDamages.Get(c.E)
		if !ok {
			continue
		}

		if !cd.IsActive() {
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

		if cd.CanHitTile(c.Tile) && ecsCont.Terrain().DamageTile(c.Tile, damageTiers[c.ShapeIdx]) {
			cd.RecordTileHit(c.Tile)
			if err := effects.SpawnDamageNumber(utils.TilePosToWorldPos(c.Tile), damageTiers[c.ShapeIdx], ecsCont); err != nil {
				log.Println(err)
			}
		}
	}
}
