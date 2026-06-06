package commonsystems

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"log"
	"slices"
)

func PopulateSpatialHashGrid(
	world *ecs.World,
) (map[common.CellKey][]common.EntityId, error) {
	grid := make(map[common.CellKey][]common.EntityId)
	tm := ecs.TransformManager{}

	for e, _ := range world.Transforms {
		worldPos, err := tm.GetWorldPos(e, world.Transforms, world.Parents)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", e, err)
			continue
		}

		x := int(worldPos.X / data.SpatialHashGridCellSize)
		y := int(worldPos.Y / data.SpatialHashGridCellSize)
		grid[common.CellKey{X: x, Y: y}] = append(grid[common.CellKey{X: x, Y: y}], e)
	}

	return grid, nil
}

func GetSHGProximities(
	shg map[common.CellKey][]common.EntityId,
	world *ecs.World,
) (map[common.EntityId][]common.EntityId, error) {
	proximateEntities := make(map[common.EntityId][]common.EntityId)

	for eA, _ := range world.Transforms {
		tm := ecs.TransformManager{}
		worldPosA, err := tm.GetWorldPos(eA, world.Transforms, world.Parents)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", eA, err)
			continue
		}

		cellX := int(worldPosA.X / data.SpatialHashGridCellSize)
		cellY := int(worldPosA.Y / data.SpatialHashGridCellSize)
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for _, eB := range shg[common.CellKey{X: cellX + dx, Y: cellY + dy}] {
					if eA == eB {
						continue
					}

					_, ok := world.Transforms[eB]
					if !ok {
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
