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

type ephemeralTileDto struct {
	GridPos utils.CellKey
}

func (ephemeralTileDto) isComponentDto() {}

func (x ephemeralTile) ToDto() ephemeralTileDto {
	return ephemeralTileDto{
		GridPos: x.gridPos,
	}
}

func (x ephemeralTileDto) ToComponent() *ephemeralTile {
	return &ephemeralTile{
		gridPos: x.GridPos,
	}
}
