package tilesystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

func (cc *ChunkContainer) GetRequiredChunks(
	ecsContainer *ecs.ECSContainer,
) (toBeAdded []utils.CellKey, toBeRemoved []utils.CellKey, err error) {
	chunks := make(map[utils.CellKey]struct{})

	for _, e := range ecsContainer.ChunkLoaders.GetEntities() {
		if !ecsContainer.Transforms.HasComponent(e) {
			return nil, nil, &common.ErrorMissingComponentDependency{
				Entity:           e,
				PresentComponent: "ChunkLoader",
				MissingComponent: "Transform",
			}
		}

		eWorldPos, err := ecsContainer.TransformManager.GetWorldPos(e, ecsContainer)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting world position of entity %d: %v", e, err)
		}

		clRadius, err := ecsContainer.ChunkLoaderManager.GetRadius(e, ecsContainer)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting radius of entity %d: %v", e, err)
		}

		chunkPos := utils.WorldPosToChunkGridPos(eWorldPos)

		for dx := -clRadius; dx <= clRadius; dx++ {
			for dy := -clRadius; dy <= clRadius; dy++ {
				requiredChunkPos := utils.CellKey{
					X: chunkPos.X + dx,
					Y: chunkPos.Y + dy,
				}
				chunks[requiredChunkPos] = struct{}{}
			}
		}
	}

	for requiredChunkPos := range chunks {
		if _, exists := cc.Chunks[requiredChunkPos]; !exists {
			toBeAdded = append(toBeAdded, requiredChunkPos)
		}
	}

	for existingChunkPos := range cc.Chunks {
		if _, exists := chunks[existingChunkPos]; !exists {
			toBeRemoved = append(toBeRemoved, existingChunkPos)
		}
	}

	return toBeAdded, toBeRemoved, nil
}
