package tilesystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

func (cc *ChunkContainer) getEntityIdsInChunks(ecsCont *ecs.ECSContainer) (
	eIdsInChunks map[utils.Vec2i][]common.EntityId,
	toBeUnloaded map[utils.Vec2i][]common.EntityId,
	err error,
) {
	tm := ecsCont.TransformManager
	pm := ecsCont.HierarchyManager

	eIdsInChunks = make(map[utils.Vec2i][]common.EntityId)
	toBeUnloaded = make(map[utils.Vec2i][]common.EntityId)

	for _, e := range ecsCont.Transforms.GetEntities() {
		eWorldPos, err := tm.GetWorldPos(e, ecsCont)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting world position of entity %d: %v", e, err)
		}

		eChunkPos := utils.WorldPosToChunkGridPos(eWorldPos)
		eIdsInChunks[eChunkPos] = append(eIdsInChunks[eChunkPos], e)

		if _, ok := cc.chunks[eChunkPos]; !ok {
			// Don't queue unload for entities that have a parent,
			// they should be unloaded when the parent is.
			if pm.GetParent(e, ecsCont) == -1 {
				toBeUnloaded[eChunkPos] = append(toBeUnloaded[eChunkPos], e)
			}
		}
	}

	return eIdsInChunks, toBeUnloaded, nil
}
