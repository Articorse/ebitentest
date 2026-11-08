package data

//go:generate stringer -type=TileEnum
type TileEnum uint16

const (
	Tile_None TileEnum = iota
	Tile_Dirt
	Tile_Grass
	Tile_Rock
	Tile_WaterShallow
	Tile_WaterDeep
	Tile_WoodWall
	Tile_RockFloor
)

// A tile is made of layers, drawn and built from the bottom up
type TileLayerEnum uint8

const (
	// What the world is made of, such as dirt, rock floor or water, which is always there
	TileLayer_Ground TileLayerEnum = iota
	// Built on the ground to make it usable, such as a bridge over water
	TileLayer_Foundation
	// What grows or is laid over the ground, such as grass or a floor, which walls clear
	TileLayer_Cover
	TileLayer_Wall
	TileLayerCount
)

// The tile on each layer, Tile_None where there's none
type TileLayers [TileLayerCount]TileEnum

// The highest layer with a tile, which is the ground when there's nothing on it
func (x TileLayers) TopLayer() TileLayerEnum {
	for l := TileLayerCount - 1; l > TileLayer_Ground; l-- {
		if x[l] != Tile_None {
			return l
		}
	}
	return TileLayer_Ground
}

func (x TileLayers) Top() TileEnum {
	return x[x.TopLayer()]
}
