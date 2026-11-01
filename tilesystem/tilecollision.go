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

type TileCollision struct {
	E      common.EntityId
	Vector utils.Vec2f
}

func ResolveTileCollisions(
	collisions []TileCollision,
	ecsContainer *ecs.ECSContainer,
) (collisionsResolved uint64, err error) {
	for _, c := range collisions {
		t, traOk := ecsContainer.Transforms.Get(c.E)
		v, velOk := ecsContainer.Velocities.Get(c.E)
		if !traOk || !velOk {
			log.Printf("Error resolving tile collision of entity %d: it needs a transform and a velocity\n", c.E)
			continue
		}

		t.AddLocalPos(c.Vector)

		impulse := utils.CalculateImpulse(v.LocalVector(), c.Vector.Normalized(), data.Bounciness)
		if impulse.Length() > 0 {
			v.SetLocalVector(v.LocalVector().Add(impulse))

			collisionsResolved++
		}
	}

	return collisionsResolved, nil
}

func GetCollisions(
	potentialCollisions []EntityTiles,
	ecsContainer *ecs.ECSContainer,
) ([]TileCollision, error) {
	collisions := []TileCollision{}

	for _, et := range potentialCollisions {
		pc, ok := ecsContainer.PhysicsColliders.Get(et.E)
		if !ok {
			log.Printf("Error getting collider shapes for entity %d: it has no physics collider\n", et.E)
			continue
		}

		var total utils.Vec2f
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

			for _, eShape := range pc.Shapes() {
				if v := collisionsystem.ShapeCollision(-1, et.E, tShape, eShape, ecsContainer); !v.IsZero() {
					total = total.Add(v)
					break
				}
			}
		}

		if !total.IsZero() {
			collisions = append(collisions, TileCollision{E: et.E, Vector: total})
		}
	}

	return collisions, nil
}

func GetAABBCollisions(
	potentialCollisions []EntityTiles,
	ecsContainer *ecs.ECSContainer,
) ([]EntityTiles, error) {
	collisions := []EntityTiles{}

	for _, et := range potentialCollisions {
		pc, ok := ecsContainer.PhysicsColliders.Get(et.E)
		if !ok || !pc.Enabled() || pc.Mask()&ecs.Layer_Terrain == 0 {
			continue
		}

		eAABB, err := ecsContainer.WorldAABB(et.E, pc.LocalPaddedAABB())
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

func (cc *ChunkContainer) GetTilesWithPotentialCollisions(
	ecsContainer *ecs.ECSContainer,
	tileSize int,
) (potentialCollisions []EntityTiles, err error) {

	for _, e := range ecsContainer.Transforms.GetEntities() {
		pc, ok := ecsContainer.PhysicsColliders.Get(e)
		if !ok || pc.ColliderType() != ecs.Collider_Mob {
			continue
		}

		worldAABB, err := ecsContainer.WorldAABB(e, pc.LocalAABB())
		if err != nil {
			log.Printf("error getting world AABB of entity %d: %v", e, err)
			continue
		}

		tiles := []utils.Vec2i{}
		minTileX := int(math.Floor(worldAABB[0].X/float64(tileSize))) - 1
		minTileY := int(math.Floor(worldAABB[0].Y/float64(tileSize))) - 1
		maxTileX := int(math.Floor(worldAABB[1].X/float64(tileSize))) + 1
		maxTileY := int(math.Floor(worldAABB[1].Y/float64(tileSize))) + 1

		for tx := minTileX; tx <= maxTileX; tx++ {
			for ty := minTileY; ty <= maxTileY; ty++ {
				worldTilePos := utils.Vec2i{X: tx, Y: ty}
				chunkGridPos := utils.WorldPosToChunkGridPos(utils.Vec2f{X: float64(worldTilePos.X * tileSize), Y: float64(worldTilePos.Y * tileSize)})
				localTilePos := utils.Vec2i{
					X: ((worldTilePos.X % int(data.ChunkSize)) + int(data.ChunkSize)) % int(data.ChunkSize),
					Y: ((worldTilePos.Y % int(data.ChunkSize)) + int(data.ChunkSize)) % int(data.ChunkSize),
				}

				chunk, ok := cc.chunks[chunkGridPos]
				if !ok {
					tiles = append(tiles, worldTilePos)
					continue
				}

				tileId := chunk.getTileDefId(localTilePos)
				tileDef, ok := cc.atlas[tileId]
				if !ok {
					log.Printf("no tile definition found for tile enum %d", tileId)
					continue
				}

				if !tileDef.passable {
					tiles = append(tiles, worldTilePos)
				}
			}
		}

		if len(tiles) > 0 {
			potentialCollisions = append(potentialCollisions, EntityTiles{E: e, Tiles: tiles})
		}
	}

	return potentialCollisions, nil
}
