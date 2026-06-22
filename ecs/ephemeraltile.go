package ecs

import "ebittest/ecs/common"

type ephemeralTile struct {
	gridPos common.CellKey
}

func (ephemeralTile) isComponent() {}

func (x ephemeralTile) Copy() ephemeralTile {
	return ephemeralTile{
		gridPos: x.gridPos,
	}
}
