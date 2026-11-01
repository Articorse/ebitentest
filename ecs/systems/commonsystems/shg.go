package commonsystems

import (
	"cmp"
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
) (map[utils.Vec2i][]common.EntityId, error) {
	grid := make(map[utils.Vec2i][]common.EntityId)

	for _, e := range ecsContainer.Transforms.GetEntities() {
		worldPos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", e, err)
			continue
		}

		x := int(worldPos.X / float64(cellSize))
		y := int(worldPos.Y / float64(cellSize))
		grid[utils.Vec2i{X: x, Y: y}] = append(grid[utils.Vec2i{X: x, Y: y}], e)
	}

	return grid, nil
}

// Every pair is listed once, lower ID first, in ID order
func GetSHGProximities(
	shg map[utils.Vec2i][]common.EntityId,
	ecsContainer *ecs.ECSContainer,
) ([]common.EntityPair, error) {
	proximateEntities := []common.EntityPair{}

	for _, eA := range ecsContainer.Transforms.GetEntities() {
		worldPosA, err := ecsContainer.WorldPos(eA)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", eA, err)
			continue
		}

		start := len(proximateEntities)
		cellX := int(worldPosA.X / data.SpatialHashGridCellSize)
		cellY := int(worldPosA.Y / data.SpatialHashGridCellSize)
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for _, eB := range shg[utils.Vec2i{X: cellX + dx, Y: cellY + dy}] {
					if eB <= eA {
						continue
					}

					if !ecsContainer.Transforms.HasComponent(eB) {
						return nil, &common.ErrorMissingComponentDependency{
							Entity:           eB,
							PresentComponent: "Collider",
							MissingComponent: "Transform",
						}
					}

					proximateEntities = append(proximateEntities, common.EntityPair{A: eA, B: eB})
				}
			}
		}

		slices.SortFunc(proximateEntities[start:], func(p, q common.EntityPair) int { return cmp.Compare(p.B, q.B) })
	}

	return proximateEntities, nil
}
