package commonsystems

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"slices"
)

func PopulateSpatialHashGrid(
	transforms map[ecscommon.EntityId]*components.Transform,
) (map[ecscommon.CellKey][]ecscommon.EntityId, error) {
	grid := make(map[ecscommon.CellKey][]ecscommon.EntityId)
	for e, tra := range transforms {
		x := int(tra.Pos.X / data.SpatialHashGridCellSize)
		y := int(tra.Pos.Y / data.SpatialHashGridCellSize)
		grid[ecscommon.CellKey{X: x, Y: y}] = append(grid[ecscommon.CellKey{X: x, Y: y}], e)
	}

	return grid, nil
}

func GetSHGProximities[T components.Component](
	shg map[ecscommon.CellKey][]ecscommon.EntityId,
	requiredComponentMap map[ecscommon.EntityId]*T,
	transforms map[ecscommon.EntityId]*components.Transform,
) (map[ecscommon.EntityId][]ecscommon.EntityId, error) {
	proximateEntities := make(map[ecscommon.EntityId][]ecscommon.EntityId)

	for eA, _ := range requiredComponentMap {
		traA, ok := transforms[eA]
		if !ok {
			return nil, &ecscommon.ErrorMissingComponentDependency{
				Entity:           eA,
				PresentComponent: "Collider",
				MissingComponent: "Transform",
			}
		}
		cellX := int(traA.Pos.X / data.SpatialHashGridCellSize)
		cellY := int(traA.Pos.Y / data.SpatialHashGridCellSize)
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
