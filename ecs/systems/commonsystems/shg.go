package commonsystems

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"log"
	"slices"
)

func PopulateSpatialHashGrid(
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) (map[ecscommon.CellKey][]ecscommon.EntityId, error) {
	grid := make(map[ecscommon.CellKey][]ecscommon.EntityId)
	tm := components.TransformManager{}

	for e, _ := range transforms {
		worldPos, err := tm.GetWorldPos(e, transforms, parents)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", e, err)
			continue
		}

		x := int(worldPos.X / data.SpatialHashGridCellSize)
		y := int(worldPos.Y / data.SpatialHashGridCellSize)
		grid[ecscommon.CellKey{X: x, Y: y}] = append(grid[ecscommon.CellKey{X: x, Y: y}], e)
	}

	return grid, nil
}

func GetSHGProximities[T components.Component](
	shg map[ecscommon.CellKey][]ecscommon.EntityId,
	requiredComponentMap map[ecscommon.EntityId]*T,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) (map[ecscommon.EntityId][]ecscommon.EntityId, error) {
	proximateEntities := make(map[ecscommon.EntityId][]ecscommon.EntityId)

	for eA, _ := range requiredComponentMap {
		tm := components.TransformManager{}
		worldPosA, err := tm.GetWorldPos(eA, transforms, parents)
		if err != nil {
			log.Printf("error getting world position of entity %d: %v", eA, err)
			continue
		}

		cellX := int(worldPosA.X / data.SpatialHashGridCellSize)
		cellY := int(worldPosA.Y / data.SpatialHashGridCellSize)
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for _, eB := range shg[ecscommon.CellKey{X: cellX + dx, Y: cellY + dy}] {
					if eA == eB {
						continue
					}

					_, ok := requiredComponentMap[eB]
					if !ok {
						continue
					}

					_, ok = transforms[eB]
					if !ok {
						return nil, &ecscommon.ErrorMissingComponentDependency{
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
