package commonsystems

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
	"slices"
)

func PopulateSpatialHashGrid(
	ecsContainer *ecs.ECSContainer,
	cellSize int,
) (map[utils.CellKey][]common.EntityId, error) {
	grid := make(map[utils.CellKey][]common.EntityId)
	tm := ecsContainer.TransformManager

	for _, e := range ecsContainer.Transforms.GetEntities() {
		worldPos, err := tm.GetWorldPos(e, ecsContainer)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", e, err)
			continue
		}

		x := int(worldPos.X / float64(cellSize))
		y := int(worldPos.Y / float64(cellSize))
		grid[utils.CellKey{X: x, Y: y}] = append(grid[utils.CellKey{X: x, Y: y}], e)
	}

	return grid, nil
}

func GetSHGProximities(
	shg map[utils.CellKey][]common.EntityId,
	ecsContainer *ecs.ECSContainer,
) (map[common.EntityId][]common.EntityId, error) {
	proximateEntities := make(map[common.EntityId][]common.EntityId)

	for _, eA := range ecsContainer.Transforms.GetEntities() {
		tm := ecsContainer.TransformManager
		worldPosA, err := tm.GetWorldPos(eA, ecsContainer)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", eA, err)
			continue
		}

		cellX := int(worldPosA.X / data.SpatialHashGridCellSize)
		cellY := int(worldPosA.Y / data.SpatialHashGridCellSize)
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for _, eB := range shg[utils.CellKey{X: cellX + dx, Y: cellY + dy}] {
					if eA == eB {
						continue
					}

					if !ecsContainer.Transforms.HasComponent(eB) {
						return nil, &common.ErrorMissingComponentDependency{
							Entity:           eB,
							PresentComponent: "Collider",
							MissingComponent: "Transform",
						}
					}

					if proximateEntity, ok := proximateEntities[eB]; ok {
						if slices.Contains(proximateEntity, eA) {
							continue
						}
					}

					if !slices.Contains(proximateEntities[eA], eB) {
						proximateEntities[eA] = append(proximateEntities[eA], eB)
					}
				}
			}
		}
	}

	return proximateEntities, nil
}
