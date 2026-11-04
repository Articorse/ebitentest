package data

const (
	// How many tiles a path search may look at before giving up, which bounds what one search costs
	PathMaxExpandedTiles = 10000
	// How far along its path something walking it looks for a tile it can head straight to, so that it doesn't zigzag
	// from tile center to tile center
	PathLookaheadTiles = 8
)
