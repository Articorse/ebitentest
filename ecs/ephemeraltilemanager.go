package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type ephemeralTileManager struct{}

func NewEphemeralTileComponent(gridPos utils.CellKey) *ephemeralTile {
	return &ephemeralTile{
		gridPos: gridPos,
	}
}

func (ephemeralTileManager) GetEntityIdByGridPos(
	gridPos utils.CellKey,
	ecsContainer *ECSContainer,
) (common.EntityId, error) {
	for _, e := range ecsContainer.EphemeralTiles.GetEntities() {
		etComp, err := ecsContainer.EphemeralTiles.getComponent(e)
		if err != nil {
			continue
		}

		if etComp.gridPos == gridPos {
			return e, nil
		}
	}

	return -1, fmt.Errorf("no ephemeral tile found at grid position %v", gridPos)
}
