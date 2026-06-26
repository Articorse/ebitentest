package ecs

import "ebittest/utils"

type ephemeralTile struct {
	gridPos utils.CellKey
}

func (ephemeralTile) isComponent() {}

func (x ephemeralTile) Copy() ephemeralTile {
	return ephemeralTile{
		gridPos: x.gridPos,
	}
}
