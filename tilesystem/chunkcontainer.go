package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"slices"
)

type ChunkContainer struct {
	chunks [1]chunk

	Atlas map[data.TileEnum]TileDef
}

func (x *ChunkContainer) Generate(r *rand.Rand) error {
	err := x.generateTileAtlasFromJson("assets/tiles/atlas.json")
	if err != nil {
		return fmt.Errorf("failed to generate tile atlas: %v", err)
	}
	for i := range x.chunks {
		if err = x.chunks[i].newChunk(r, x.Atlas); err != nil {
			return fmt.Errorf("failed to generate chunk %d: %v", i, err)
		}
	}
	return nil
}

func (x *ChunkContainer) GetChunks() [1]chunk {
	return x.chunks
}

func (x *ChunkContainer) generateTileAtlasFromJson(input string) error {
	f, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("error reading tile atlas json file: %v", err)
	}

	var tileDefs *[]tileDefDTO
	err = json.Unmarshal(f, &tileDefs)
	if err != nil {
		return fmt.Errorf("error unmarshalling tile atlas json: %v", err)
	}

	atlas, err := dtosToDefsMap(*tileDefs)
	if err != nil {
		return fmt.Errorf("error converting tile atlas DTOs to defs: %v", err)
	}

	x.Atlas = atlas

	return nil
}

func (x *ChunkContainer) PopulateEphemeralColliders(tileKeys []common.CellKey, ecsContainer *ecs.ECSContainer) error {
	etm := ecsContainer.EphemeralTileManager
	currentColliderEntities := []common.EntityId{}

	for _, tileKey := range tileKeys {
		tileDefId, exists := x.chunks[0].GetTileDefId(tileKey)
		if !exists {
			continue
		}

		tileDef, exists := x.Atlas[tileDefId]
		if !exists {
			log.Printf("no tile definition found for tile enum %d", tileDefId)
			continue
		}

		if !tileDef.Passable {
			existingE, _ := etm.GetEntityIdByGridPos(tileKey, ecsContainer)

			if existingE != -1 {
				currentColliderEntities = append(currentColliderEntities, existingE)
				continue
			}

			traComp := ecs.NewTransformComponent(
				utils.Vec2{X: float64(tileKey.X * data.TileSize), Y: float64(tileKey.Y * data.TileSize)}, // TODO: Add Chunk position offset
				1,
				0,
			)

			shape, err := shapes.NewRectangleShape(data.TileSize, data.TileSize, utils.Vec2{})
			if err != nil {
				log.Printf("error creating rectangle shape for tile at grid position %v: %v", tileKey, err)
				continue
			}

			phcComp := ecs.NewPhysicsColliderComponent(
				ecs.Layer_Terrain,
				ecs.Layer_Player|ecs.Layer_Enemy|ecs.Layer_FriendlyProjectile|ecs.Layer_EnemyProjectile,
				ecs.Collider_Static,
				shape,
			)

			etComp := ecs.NewEphemeralTileComponent(tileKey)

			e := ecsContainer.AddEntity(traComp, phcComp, etComp)
			currentColliderEntities = append(currentColliderEntities, e)
		}
	}

	ephemeralEntities := ecsContainer.EphemeralTiles.GetEntities()
	for _, e := range ephemeralEntities {
		if !slices.Contains(currentColliderEntities, e) {
			ecsContainer.ScheduleRemoveEntity(e)
		}
	}

	return nil
}

func (x *ChunkContainer) GetTilesWithPotentialCollisions(
	ecsContainer *ecs.ECSContainer,
	tileSize int,
) (tiles []common.CellKey, err error) {
	pcm := ecsContainer.PhysicsColliderManager

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
				tile := common.CellKey{X: tx, Y: ty}
				tiles = append(tiles, tile)
			}
		}
	}

	return tiles, nil
}
