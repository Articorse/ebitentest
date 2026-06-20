package commonsystems

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"log"
	"slices"
)

func PopulateSpatialHashGrid(
	ecs *ecs.ECS,
) (map[common.CellKey][]common.EntityId, error) {
	grid := make(map[common.CellKey][]common.EntityId)
	tm := ecs.TransformManager

	for _, e := range ecs.Transforms.GetEntities() {
		ecsPos, err := tm.GetWorldPos(e, ecs)
		if err != nil {
			log.Printf("error getting ecs position of entity %d: %v", e, err)
			continue
		}

		x := int(ecsPos.X / data.SpatialHashGridCellSize)
		y := int(ecsPos.Y / data.SpatialHashGridCellSize)
		grid[common.CellKey{X: x, Y: y}] = append(grid[common.CellKey{X: x, Y: y}], e)
	}

	return grid, nil
}

func GetSHGProximities(
	shg map[common.CellKey][]common.EntityId,
	ecs *ecs.ECS,
) (map[common.EntityId][]common.EntityId, error) {
	proximateEntities := make(map[common.EntityId][]common.EntityId)

	for _, eA := range ecs.Transforms.GetEntities() {
		tm := ecs.TransformManager
		ecsPosA, err := tm.GetWorldPos(eA, ecs)
		if err != nil {
			log.Printf("error getting ecs position of entity %d: %v", eA, err)
			continue
		}

		cellX := int(ecsPosA.X / data.SpatialHashGridCellSize)
		cellY := int(ecsPosA.Y / data.SpatialHashGridCellSize)
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for _, eB := range shg[common.CellKey{X: cellX + dx, Y: cellY + dy}] {
					if eA == eB {
						continue
					}

					if !ecs.Transforms.HasComponent(eB) {
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
