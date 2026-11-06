package data

import "image/color"

var (
	// Premultiplied, and faint, since an area can cover much of the screen
	HomeAreaColor = color.RGBA{R: 20, G: 60, B: 20, A: 60}
	// Yellow-orange
	HarvestAreaColor = color.RGBA{R: 70, G: 45, B: 5, A: 70}
	// The rectangle being dragged out, marking or unmarking
	HomeMarkColor   = color.RGBA{R: 30, G: 110, B: 30, A: 110}
	AreaUnmarkColor = color.RGBA{R: 110, G: 25, B: 25, A: 110}
	// Yellow-orange
	HarvestMarkColor = color.RGBA{R: 120, G: 80, B: 10, A: 120}
	// The squares on the area menu's buttons, opaque so they read as the area's colour against the slot
	HomeButtonColor    = color.RGBA{R: 60, G: 170, B: 60, A: 255}
	HarvestButtonColor = color.RGBA{R: 230, G: 160, B: 30, A: 255}
)

const (
	// The middle of the chunk at the world's origin, so that the colony's bounds, which are whole chunks, are centered
	// on it
	ColonyCenterTileX, ColonyCenterTileY = ChunkSize / 2, ChunkSize / 2
	// The chunks around the center's own that are the colony, so 1 is 3×3 chunks
	ColonyRadiusChunks = 1
	// How far around something built on home the home spreads, as a square
	HomeSpreadTiles = 15

	ColonistInventorySlots = 9
	ColonistStartingWood   = 20
	// How often a colonist looks for work while it has none, and for building while it's delivering, since looking can
	// mean searching the whole colony
	ColonistPlanEveryTicks = TPS
	// How many of the sites closest as the crow flies it finds paths to, choosing the one cheapest to walk to
	ColonistJobCandidates = 3
	// Chopping is done from next to a tree, not diagonally, where an axe swung from the far side of the tile can fall
	// short of it
	ColonistChopRangeTiles = 1
)
