package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/utils"
	"log"
	"math"
)

type EntityTiles struct {
	E     common.EntityId
	Tiles []utils.Vec2i
}

// Pushes each entity out by the sum of its tile collisions, which are grouped by entity
func ResolveTileCollisions(
	collisions []common.TileCollision,
	ecsContainer *ecs.ECSContainer,
) (collisionsResolved uint64, err error) {
	for i := 0; i < len(collisions); {
		e := collisions[i].E
		var total utils.Vec2f
		for ; i < len(collisions) && collisions[i].E == e; i++ {
			total = total.Add(collisions[i].Vector)
		}

		t, traOk := ecsContainer.Transforms.Get(e)
		v, velOk := ecsContainer.Velocities.Get(e)
		if !traOk || !velOk {
			log.Printf("Error resolving tile collision of entity %d: it needs a transform and a velocity\n", e)
			continue
		}

		t.AddLocalPos(total)

		impulse := utils.CalculateImpulse(v.LocalVector(), total.Normalized(), data.Bounciness)
		if impulse.Length() > 0 {
			v.SetLocalVector(v.LocalVector().Add(impulse))

			collisionsResolved++
		}
	}

	return collisionsResolved, nil
}

// In the order of the potential collisions, with the first of the entity's shapes that overlaps each tile
func GetCollisions(
	colliders ecs.ColliderSet,
	potentialCollisions []EntityTiles,
	ecsContainer *ecs.ECSContainer,
) ([]common.TileCollision, error) {
	collisions := []common.TileCollision{}

	for _, et := range potentialCollisions {
		col, ok := colliders.Collider(et.E)
		if !ok {
			log.Printf("Error getting collider shapes for entity %d: it has no collider\n", et.E)
			continue
		}

		for _, t := range et.Tiles {
			tShape, err := shapes.NewRectangleShape(
				data.TileSize,
				data.TileSize,
				utils.Vec2f{
					X: float64(t.X) * data.TileSize,
					Y: float64(t.Y) * data.TileSize,
				},
			)
			if err != nil {
				log.Printf("Error creating rectangle shape for tile at grid position %v: %v\n", t, err)
				continue
			}

			for shapeIdx, eShape := range col.Shapes() {
				if v := collisionsystem.ShapeCollision(-1, et.E, tShape, eShape, ecsContainer); !v.IsZero() {
					collisions = append(collisions, common.TileCollision{E: et.E, Tile: t, Vector: v, ShapeIdx: shapeIdx})
					break
				}
			}
		}
	}

	return collisions, nil
}

func GetAABBCollisions(
	colliders ecs.ColliderSet,
	potentialCollisions []EntityTiles,
	ecsContainer *ecs.ECSContainer,
) ([]EntityTiles, error) {
	collisions := []EntityTiles{}

	for _, et := range potentialCollisions {
		col, ok := colliders.Collider(et.E)
		if !ok {
			continue
		}

		eAABB, err := ecsContainer.WorldAABB(et.E, col.LocalPaddedAABB())
		if err != nil {
			log.Printf("Error getting AABB for entity %d: %v\n", et.E, err)
			continue
		}

		collided := EntityTiles{E: et.E}
		for _, t := range et.Tiles {
			tAABB := [2]utils.Vec2f{
				{X: (float64(t.X) - 0.5) * data.TileSize, Y: (float64(t.Y) - 0.5) * data.TileSize},
				{X: (float64(t.X) + 0.5) * data.TileSize, Y: (float64(t.Y) + 0.5) * data.TileSize},
			}

			if utils.DetectAABBCollision(eAABB, tAABB) {
				collided.Tiles = append(collided.Tiles, t)
			}
		}

		if len(collided.Tiles) > 0 {
			collisions = append(collisions, collided)
		}
	}

	return collisions, nil
}

// Unloaded tiles count as impassable, so that nothing walks out of the simulated area
func (cc *ChunkContainer) GetTilesWithPotentialPhysicsCollisions(ecsContainer *ecs.ECSContainer) ([]EntityTiles, error) {
	return cc.getTilesWithPotentialCollisions(ecs.Colliders(&ecsContainer.PhysicsColliders), true, ecsContainer, func(e common.EntityId) bool {
		pc, _ := ecsContainer.PhysicsColliders.Get(e)
		return pc.ColliderType() == ecs.Collider_Mob
	})
}

// Only tiles that are really impassable count, so that a projectile leaving the simulated area doesn't hit its edge
func (cc *ChunkContainer) GetHurtboxTileCollisions(ecsContainer *ecs.ECSContainer) ([]common.TileCollision, error) {
	hurtboxes := ecs.Colliders(&ecsContainer.HurtboxColliders)

	potential, err := cc.getTilesWithPotentialCollisions(hurtboxes, false, ecsContainer, nil)
	if err != nil {
		return nil, err
	}

	aabb, err := GetAABBCollisions(hurtboxes, potential, ecsContainer)
	if err != nil {
		return nil, err
	}

	return GetCollisions(hurtboxes, aabb, ecsContainer)
}

func (cc *ChunkContainer) getTilesWithPotentialCollisions(
	colliders ecs.ColliderSet,
	unloadedImpassable bool,
	ecsContainer *ecs.ECSContainer,
	include func(common.EntityId) bool,
) (potentialCollisions []EntityTiles, err error) {
	for _, e := range colliders.GetEntities() {
		col, _ := colliders.Collider(e)
		if !col.Enabled() || col.Mask()&ecs.Layer_Terrain == 0 || (include != nil && !include(e)) {
			continue
		}

		worldAABB, err := ecsContainer.WorldAABB(e, col.LocalAABB())
		if err != nil {
			log.Printf("error getting world AABB of entity %d: %v", e, err)
			continue
		}

		tiles := []utils.Vec2i{}
		minTileX := int(math.Floor(worldAABB[0].X/data.TileSize)) - 1
		minTileY := int(math.Floor(worldAABB[0].Y/data.TileSize)) - 1
		maxTileX := int(math.Floor(worldAABB[1].X/data.TileSize)) + 1
		maxTileY := int(math.Floor(worldAABB[1].Y/data.TileSize)) + 1

		for tx := minTileX; tx <= maxTileX; tx++ {
			for ty := minTileY; ty <= maxTileY; ty++ {
				t := utils.Vec2i{X: tx, Y: ty}

				chunk, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
				if !ok {
					if unloadedImpassable {
						tiles = append(tiles, t)
					}
					continue
				}

				tileId := chunk.getTileDefId(t)
				tileDef, ok := cc.atlas[tileId]
				if !ok {
					log.Printf("no tile definition found for tile enum %d", tileId)
					continue
				}

				if !tileDef.passable {
					tiles = append(tiles, t)
				}
			}
		}

		if len(tiles) > 0 {
			potentialCollisions = append(potentialCollisions, EntityTiles{E: e, Tiles: tiles})
		}
	}

	return potentialCollisions, nil
}
