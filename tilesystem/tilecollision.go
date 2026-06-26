package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/utils"
	"fmt"
	"log"
	"math"
)

func ResolveTileCollisions(
	collisions map[common.EntityId]utils.Vec2,
	ecsContainer *ecs.ECSContainer,
) (collisionsResolved uint64, err error) {
	tm := ecsContainer.TransformManager
	vm := ecsContainer.VelocityManager

	for e, c := range collisions {
		mobLocalPos, err := tm.GetLocalPos(e, ecsContainer)
		if err != nil {
			log.Printf("Error getting local position for entity %d: %v\n", e, err)
			continue
		}
		mobLocalVelVec, err := vm.GetLocalVector(e, ecsContainer)
		if err != nil {
			log.Printf("Error getting local velocity vector for entity %d: %v\n", e, err)
			continue
		}

		err = tm.SetLocalPos(e, mobLocalPos.Add(c), ecsContainer)
		if err != nil {
			log.Printf("Error setting local position for entity %d: %v\n", e, err)
			continue
		}

		impulse := utils.CalculateImpulse(mobLocalVelVec, c.Normalized(), data.Bounciness)
		if impulse.Length() > 0 {
			err = vm.SetLocalVector(e, mobLocalVelVec.Add(impulse), ecsContainer)
			if err != nil {
				log.Printf("Error setting local velocity vector for entity %d: %v\n", e, err)
				continue
			}

			collisionsResolved++
		}
	}

	return collisionsResolved, nil
}

func GetCollisions(
	potentialCollisions map[common.EntityId][]utils.CellKey,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId]utils.Vec2, error) {
	pcm := ecsContainer.PhysicsColliderManager
	collisions := make(map[common.EntityId]utils.Vec2)

	for e, colTiles := range potentialCollisions {
		for _, t := range colTiles {
			eColShapes, err := pcm.GetShapes(e, ecsContainer)
			if err != nil {
				log.Printf("Error getting collider shapes for entity %d: %v\n", e, err)
				continue
			}

			tShape, err := shapes.NewRectangleShape(
				data.TileSize,
				data.TileSize,
				utils.Vec2{
					X: float64(t.X) * data.TileSize,
					Y: float64(t.Y) * data.TileSize,
				},
			)
			if err != nil {
				log.Printf("Error creating rectangle shape for tile at grid position %v: %v\n", t, err)
				continue
			}

			collisionFound := false
			var collisionVector utils.Vec2
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
	potentialCollisions map[common.EntityId][]utils.CellKey,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId][]utils.CellKey, error) {
	pcm := ecsContainer.PhysicsColliderManager
	collisions := make(map[common.EntityId][]utils.CellKey)

	for e, colTiles := range potentialCollisions {
		for _, t := range colTiles {
			if !pcm.HasCollider(e, ecsContainer) {
				continue
			}

			pcEnabled, err := pcm.IsEnabled(e, ecsContainer)
			if err != nil {
				log.Printf("Error checking if collider is enabled for entity %d: %v\n", e, err)
				continue
			}

			if !pcEnabled {
				continue
			}

			pcMask, err := pcm.GetMask(e, ecsContainer)
			if err != nil {
				log.Printf("Error getting collider mask for entity %d: %v\n", e, err)
				continue
			}

			if pcMask&ecs.Layer_Terrain == 0 {
				continue
			}

			eAABB, err := pcm.GetWorldPaddedAABB(e, ecsContainer)
			if err != nil {
				log.Printf("Error getting AABB for entity %d: %v\n", e, err)
				continue
			}

			tAABB := [2]utils.Vec2{
				utils.Vec2{
					X: float64(t.X) * data.TileSize,
					Y: float64(t.Y) * data.TileSize,
				},
				utils.Vec2{
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
) (potentialCollisions map[common.EntityId][]utils.CellKey, err error) {
	pcm := ecsContainer.PhysicsColliderManager
	potentialCollisions = make(map[common.EntityId][]utils.CellKey)

	for _, e := range ecsContainer.Transforms.GetEntities() {
		if !pcm.HasCollider(e, ecsContainer) {
			continue
		}

		colType, err := pcm.GetColliderType(e, ecsContainer)
		if err != nil {
			log.Printf("error getting collider type of entity %d: %v", e, err)
			continue
		}

		if colType != ecs.Collider_Mob {
			continue
		}

		worldAABB, err := pcm.GetWorldAABB(e, ecsContainer)
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
				worldTilePos := utils.CellKey{X: tx, Y: ty}
				chunkGridPos := utils.WorldPosToChunkGridPos(utils.Vec2{X: float64(worldTilePos.X * tileSize), Y: float64(worldTilePos.Y * tileSize)})
				chunk, ok := cc.Chunks[chunkGridPos]
				if !ok {
					fmt.Printf("no chunk found at grid position %v for world tile position %v\n", chunkGridPos, worldTilePos)
					continue
				}

				localTilePos := utils.CellKey{
					X: ((worldTilePos.X % int(data.ChunkSize)) + int(data.ChunkSize)) % int(data.ChunkSize),
					Y: ((worldTilePos.Y % int(data.ChunkSize)) + int(data.ChunkSize)) % int(data.ChunkSize),
				}
				tileId := chunk.GetTileDefId(localTilePos)
				tileDef, ok := cc.Atlas[tileId]
				if !ok {
					log.Printf("no tile definition found for tile enum %d", tileId)
					continue
				}

				if !tileDef.Passable {
					potentialCollisions[e] = append(potentialCollisions[e], worldTilePos)
				}
			}
		}
	}

	return potentialCollisions, nil
}
