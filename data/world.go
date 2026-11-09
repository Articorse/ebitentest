package data

const (
	TileSize                    = 16
	ChunkSize                   = 128
	ChunkDataDir                = "chunkdata"
	ChunkDataFilePathTemplate   = "chunkdata/chunk_%d_%d.gob"
	WorldHeaderFilePath         = "chunkdata/world.gob"
	RoomsFilePath               = "chunkdata/rooms.gob"
	WorldFormatVersion          = 1
	WorldSavePeriodSec          = 60
	ChunkReloadRetries          = 3
	MaxChunkLoadDispatchPerTick = 4
	ChunkDeactivateMargin       = 1
	ChunkPrefetchMargin         = 2
	ChunkImageBakeMarginPx      = 100
	ChunkImageFreeMarginPx      = 300
	MaxChunkImageBakesPerFrame  = 1
	TetherBorderTiles           = 1
	TetherSearchRadiusTiles     = 16
)
