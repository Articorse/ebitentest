package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
)

type ChunkContainer struct {
	chunks map[common.CellKey]*chunk

	Atlas map[data.TileEnum]TileDef
}

func (cC *ChunkContainer) newChunk(r *rand.Rand, atlas map[data.TileEnum]TileDef) (*chunk, error) {
	c := chunk{}
	c.promotedTiles = make(map[common.CellKey]promotedTile)

	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			tileId := data.TileEnum(r.IntN(3) + 1)
			c.tiles[y*data.ChunkSize+x] = tileId
		}
	}

	if err := c.generateChunkImage(atlas); err != nil {
		return nil, fmt.Errorf("failed to generate chunk image: %v", err)
	}

	return &c, nil
}

func (x *ChunkContainer) Generate(r *rand.Rand) error {
	x.chunks = make(map[common.CellKey]*chunk)

	err := x.generateTileAtlasFromJson("assets/tiles/atlas.json")
	if err != nil {
		return fmt.Errorf("failed to generate tile atlas: %v", err)
	}
	if x.chunks[common.CellKey{}], err = x.newChunk(r, x.Atlas); err != nil {
		return fmt.Errorf("failed to generate chunk: %v", err)
	}
	return nil
}

func (x *ChunkContainer) GetChunks() map[common.CellKey]*chunk {
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

func (x *ChunkContainer) GetTilesWithPotentialCollisions(
	ecsContainer *ecs.ECSContainer,
	tileSize int,
) (potentialCollisions map[common.EntityId][]common.CellKey, err error) {
	pcm := ecsContainer.PhysicsColliderManager
	potentialCollisions = make(map[common.EntityId][]common.CellKey)

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
				tilePos := common.CellKey{X: tx, Y: ty}
				chunk, err := x.GetChunkAtGridPos(tilePos)
				if err != nil {
					log.Printf("error getting chunk at grid position %v: %v", tilePos, err)
					continue
				}

				tileId := chunk.GetTileDefId(tilePos)
				tileDef, ok := x.Atlas[tileId]
				if !ok {
					log.Printf("no tile definition found for tile enum %d", tileId)
					continue
				}

				if !tileDef.Passable {
					potentialCollisions[e] = append(potentialCollisions[e], tilePos)
				}
			}
		}
	}

	return potentialCollisions, nil
}

func (c *ChunkContainer) GetChunkAtGridPos(pos common.CellKey) (*chunk, error) {
	y := int(int(pos.Y) / data.ChunkSize)
	x := int(int(pos.X) / data.ChunkSize)

	r, ok := c.chunks[common.CellKey{X: x, Y: y}]
	if !ok {
		return nil, fmt.Errorf("no chunk found at grid position %v", pos)
	}

	return r, nil
}
