package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
)

type ChunkContainer struct {
	chunks map[utils.CellKey]*chunk

	Atlas map[data.TileEnum]TileDef
}

func (cc *ChunkContainer) newChunk(r *rand.Rand, atlas map[data.TileEnum]TileDef) (*chunk, error) {
	c := chunk{}
	c.promotedTiles = make(map[utils.CellKey]promotedTile)

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

func (cc *ChunkContainer) Tick(r *rand.Rand, toBeAdded []utils.CellKey, toBeRemoved []utils.CellKey) error {
	if cc.chunks == nil {
		cc.chunks = make(map[utils.CellKey]*chunk)
	}

	for _, cPos := range toBeRemoved {
		delete(cc.chunks, cPos)
	}
	for _, cPos := range toBeAdded {
		var err error
		if cc.chunks[cPos], err = cc.newChunk(r, cc.Atlas); err != nil {
			return fmt.Errorf("failed to generate chunk at %v: %v", cPos, err)
		}
	}
	return nil
}

func (cc *ChunkContainer) GetChunks() map[utils.CellKey]*chunk {
	return cc.chunks
}

func (cc *ChunkContainer) GenerateTileAtlasFromJson(input string) error {
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

	cc.Atlas = atlas

	return nil
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
				chunkGridPos := utils.CellKey{
					X: int(math.Floor(math.Floor(float64(worldTilePos.X) / data.ChunkSize))),
					Y: int(math.Floor(math.Floor(float64(worldTilePos.Y) / data.ChunkSize))),
				}
				chunk, ok := cc.chunks[chunkGridPos]
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
