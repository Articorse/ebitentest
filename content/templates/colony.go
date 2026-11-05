package templates

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

func SpawnColonyCenter(ecsContainer *ecs.ECSContainer) (common.EntityId, error) {
	pos := utils.TilePosToWorldPos(utils.Vec2i{X: data.ColonyCenterTileX, Y: data.ColonyCenterTileY})
	colony, err := ecsContainer.AddNewEntity(
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(pos, 1, 0),
		ecs.NewChunkLoaderComponent(data.ColonyRadiusChunks),
		ecs.NewColonyComponent(),
		ecs.NewPersistentComponent(),
	)
	if err != nil {
		return -1, fmt.Errorf("error spawning the colony center: %w", err)
	}
	return colony, nil
}
