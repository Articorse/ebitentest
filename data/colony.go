package data

const (
	// The middle of the chunk at the world's origin, so that the colony's bounds, which are whole chunks, are centered
	// on it
	ColonyCenterTileX, ColonyCenterTileY = ChunkSize / 2, ChunkSize / 2
	// The chunks around the center's own that are the colony, so 1 is 3×3 chunks
	ColonyRadiusChunks = 1

	ColonistInventorySlots = 9
	ColonistStartingWood   = 20
)
