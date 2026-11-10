package data

type RoofEnum uint8

const (
	Roof_None RoofEnum = iota
	// What worldgen puts over mountains
	Roof_Rock
	// What colonists build on the roof area, which has to join edge to edge through roofs, within RoofSupportTiles
	// steps, to a roof over a wall or door, or to a rock roof, to be held up
	Roof_Built
)

const (
	// A colonist's build work, 1 a strike, so about a second
	RoofWork         = 4
	RoofSupportTiles = 6
	// Air walled in over more tiles than this is outdoors
	RoomMaxTiles = 1000
)
