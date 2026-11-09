package data

const (
	// How many tiles a path search may look at before giving up, which bounds what one search costs
	PathMaxExpandedTiles = 10000
	// How far along its path something walking it looks for a tile it can head straight to, so that it doesn't zigzag
	// from tile center to tile center
	PathLookaheadTiles = 8
	// How long something chasing what no path reaches waits before looking again, since a search that fails looks at
	// every tile it may
	PathRetryTicks = TPS / 4
	// How far from what can't be reached something looks for the closest it can get to instead
	PathClosestTiles = 16
)
