package commonsystems

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
)

func PopulateSpatialHashGrid(
	transforms map[ecscommon.EntityId]*components.Transform,
) (map[ecscommon.CellKey][]ecscommon.EntityId, error) {
	grid := make(map[ecscommon.CellKey][]ecscommon.EntityId)
	for e, tra := range transforms {
		x := int(tra.Pos.X / data.SpatialHashGridCellSize)
		y := int(tra.Pos.X / data.SpatialHashGridCellSize)
		grid[ecscommon.CellKey{X: x, Y: y}] = append(grid[ecscommon.CellKey{X: x, Y: y}], e)
	}

	return grid, nil
}
