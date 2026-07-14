package tilesystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

func (cc *ChunkContainer) getEntityIdsInChunks(ecsCont *ecs.ECSContainer) (map[utils.Vec2i][]common.EntityId, error) {
	tm := ecsCont.TransformManager

	eIdsInChunks := make(map[utils.Vec2i][]common.EntityId)

	for _, e := range ecsCont.Transforms.GetEntities() {
		eWorldPos, err := tm.GetWorldPos(e, ecsCont)
		if err != nil {
			return nil, fmt.Errorf("error getting world position of entity %d: %v", e, err)
		}

		eChunkPos := utils.WorldPosToChunkGridPos(eWorldPos)
		eIdsInChunks[eChunkPos] = append(eIdsInChunks[eChunkPos], e)
	}

	return eIdsInChunks, nil
}
