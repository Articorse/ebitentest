package data

//go:generate stringer -type=TileEnum
type TileEnum uint16

const (
	Tile_None TileEnum = iota
	Tile_Dirt
	Tile_Grass
	Tile_WaterShallow
	Tile_WaterDeep
	Tile_Rock
)
