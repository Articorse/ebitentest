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

func ResolveTileCollisions(
	collisions map[common.EntityId]utils.Vec2f,
	ecsContainer *ecs.ECSContainer,
) (collisionsResolved uint64, err error) {
	for e, c := range collisions {
		t, traOk := ecsContainer.Transforms.Get(e)
		v, velOk := ecsContainer.Velocities.Get(e)
		if !traOk || !velOk {
			log.Printf("Error resolving tile collision of entity %d: it needs a transform and a velocity\n", e)
			continue
		}

		t.AddLocalPos(c)

		impulse := utils.CalculateImpulse(v.LocalVector(), c.Normalized(), data.Bounciness)
		if impulse.Length() > 0 {
			v.SetLocalVector(v.LocalVector().Add(impulse))

			collisionsResolved++
		}
	}

	return collisionsResolved, nil
}

func GetCollisions(
	potentialCollisions map[common.EntityId][]utils.Vec2i,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId]utils.Vec2f, error) {
	collisions := make(map[common.EntityId]utils.Vec2f)

	for e, colTiles := range potentialCollisions {
		pc, ok := ecsContainer.PhysicsColliders.Get(e)
		if !ok {
			log.Printf("Error getting collider shapes for entity %d: it has no physics collider\n", e)
			continue
		}
		eColShapes := pc.Shapes()

		for _, t := range colTiles {

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

			collisionFound := false
			var collisionVector utils.Vec2f
			for _, eColShape := range eColShapes {
				if collisionFound {
					break
				}

				switch eS := eColShape.(type) {
				case *shapes.RectangleShape:
					collisionVector = collisionsystem.GetRectangleRectangleCollision(-1, e, *tShape, *eS, ecsContainer)
					if !collisionVector.IsZero() {
						collisionFound = true
					}
				case *shapes.CircleShape:
					collisionVector = collisionsystem.GetRectangleCircleCollision(-1, e, *tShape, *eS, ecsContainer)
					if !collisionVector.IsZero() {
						collisionFound = true
					}
				case *shapes.PolygonShape:
					collisionVector = collisionsystem.GetRectanglePolygonCollision(-1, e, *tShape, *eS, ecsContainer)
					if !collisionVector.IsZero() {
						collisionFound = true
					}
				default:
					log.Printf("unsupported collider shape type for collision detection: %T", eS)
				}
			}

			if !collisionVector.IsZero() {
				collisions[e] = collisions[e].Add(collisionVector)
			}
		}
	}

	return collisions, nil
}

func GetAABBCollisions(
	potentialCollisions map[common.EntityId][]utils.Vec2i,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId][]utils.Vec2i, error) {
	collisions := make(map[common.EntityId][]utils.Vec2i)

	for e, colTiles := range potentialCollisions {
		pc, ok := ecsContainer.PhysicsColliders.Get(e)
		if !ok || !pc.Enabled() || pc.Mask()&ecs.Layer_Terrain == 0 {
			continue
		}

		for _, t := range colTiles {
			eAABB, err := ecsContainer.WorldAABB(e, pc.LocalPaddedAABB())
			if err != nil {
				log.Printf("Error getting AABB for entity %d: %v\n", e, err)
				continue
			}

			tAABB := [2]utils.Vec2f{
				utils.Vec2f{
					X: float64(t.X) * data.TileSize,
					Y: float64(t.Y) * data.TileSize,
				},
				utils.Vec2f{
					X: float64(t.X+1) * data.TileSize,
					Y: float64(t.Y+1) * data.TileSize,
				},
			}

			if utils.DetectAABBCollision(eAABB, tAABB) {
				collisions[e] = append(collisions[e], t)
			}
		}
	}

	return collisions, nil
}

func (cc *ChunkContainer) GetTilesWithPotentialCollisions(
	ecsContainer *ecs.ECSContainer,
	tileSize int,
) (potentialCollisions map[common.EntityId][]utils.Vec2i, err error) {
	potentialCollisions = make(map[common.EntityId][]utils.Vec2i)

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
					potentialCollisions[e] = append(potentialCollisions[e], worldTilePos)
					continue
				}

				tileId := chunk.getTileDefId(localTilePos)
				tileDef, ok := cc.atlas[tileId]
				if !ok {
					log.Printf("no tile definition found for tile enum %d", tileId)
					continue
				}

				if !tileDef.passable {
					potentialCollisions[e] = append(potentialCollisions[e], worldTilePos)
				}
			}
		}
	}

	return potentialCollisions, nil
}
