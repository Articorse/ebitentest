package tilesystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

// Returns the composition roots standing in each chunk, since compositions are stored and resolved as a unit.
// Chunkloaders are left out, since they're stored in the world header and never unloaded.
func (cc *ChunkContainer) getEntityIdsInChunks(ecsCont *ecs.ECSContainer) (map[utils.Vec2i][]common.EntityId, error) {
	eIdsInChunks := make(map[utils.Vec2i][]common.EntityId)

	for _, e := range ecsCont.Transforms.GetEntities() {
		if ecsCont.CompositionRoot(e) != e || ecsCont.ChunkLoaders.HasComponent(e) {
			continue
		}

		eWorldPos, err := ecsCont.WorldPos(e)
		if err != nil {
			return nil, fmt.Errorf("error getting world position of entity %d: %v", e, err)
		}

		eChunkPos := utils.WorldPosToChunkGridPos(eWorldPos)
		eIdsInChunks[eChunkPos] = append(eIdsInChunks[eChunkPos], e)
	}

	return eIdsInChunks, nil
}

func (cc *ChunkContainer) IsActiveWorldPos(worldPos utils.Vec2f) bool {
	_, ok := cc.chunks[utils.WorldPosToChunkGridPos(worldPos)]
	return ok
}
