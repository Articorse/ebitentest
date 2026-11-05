package data

import "image/color"

var (
	// Premultiplied, and faint, since home can cover much of the screen
	HomeAreaColor = color.RGBA{R: 20, G: 60, B: 20, A: 60}
	// The rectangle being dragged out, marking or unmarking
	HomeMarkColor   = color.RGBA{R: 30, G: 110, B: 30, A: 110}
	HomeUnmarkColor = color.RGBA{R: 110, G: 25, B: 25, A: 110}
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
	// The whole colony, so that work anywhere in it is found
	ColonistSearchMaxTiles = ((2*ColonyRadiusChunks + 1) * ChunkSize) * ((2*ColonyRadiusChunks + 1) * ChunkSize)
)
