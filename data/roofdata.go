package data

type RoofEnum uint8

const (
	Roof_None RoofEnum = iota
	// What worldgen puts over mountains
	Roof_Rock
	// What colonists build on the roof area, which needs a wall or door within RoofSupportTiles to hold it up
	Roof_Built
)

const (
	// A colonist's build work, 1 a strike, so about a second
	RoofWork         = 4
	RoofSupportTiles = 6
	// Air walled in over more tiles than this is outdoors
	RoomMaxTiles = 1000
)
