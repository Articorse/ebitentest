package data

const (
	WorldgenGroundScaleTiles   = 24.0
	WorldgenGrassThreshold     = 0.0
	WorldgenRockCellSizeTiles  = 48.0
	WorldgenRockCellChance     = 0.15
	WorldgenRockWarpTiles      = 8.0
	WorldgenRockWarpScaleTiles = 20.0
	// Tunnels run along where the cave noise is close to zero, and widen where the chamber noise is high
	WorldgenCaveScaleTiles         = 40.0
	WorldgenCaveTunnelWidth        = 0.12
	WorldgenCaveChamberScaleTiles  = 24.0
	WorldgenCaveChamberThreshold   = 0.2
	WorldgenCaveChamberWidth       = 0.6
	WorldgenSpawnSearchRadiusTiles = 64
	WorldgenForestScaleTiles       = 24.0
	WorldgenForestEdge             = 0.0
	WorldgenForestFadeWidth        = 0.25
	WorldgenForestMaxTreeChance    = 1.0
	// Each lake cell may hold one lake of deep water. Radii are spread evenly on a log scale, so most lakes are small
	// and a few are huge. The cell has to be bigger than the biggest lake with its shore and wobble, since only the
	// cells around a tile's own are looked at.
	WorldgenLakeCellSizeTiles = 128.0
	WorldgenLakeChance        = 0.8
	WorldgenLakeMinRadius     = 2.0
	WorldgenLakeMaxRadius     = 45.0
	// How far the edge of a lake or pond strays from a circle, as a share of its radius, and how big its bays and points are,
	// also as a share of its radius, so that small and big lakes are the same shape
	WorldgenWaterWobble      = 0.35
	WorldgenWaterWobbleScale = 0.8
	// Deep water is surrounded by shallow water whose width wanders between these along the shore
	WorldgenShoreMinWidth   = 1.0
	WorldgenShoreMaxWidth   = 10.0
	WorldgenShoreScaleTiles = 20.0
	// Ponds are shallow water only, in cells of their own
	WorldgenPondCellSizeTiles = 40.0
	WorldgenPondChance        = 0.25
	WorldgenPondMinRadius     = 1.5
	WorldgenPondMaxRadius     = 5.0
)
