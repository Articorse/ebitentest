package data

const (
	MemoryRegionSize       = 32
	MemoryFilePrefix       = "memory_"
	MemoryFilePathTemplate = "chunkdata/memory_%d_%d.gob"
	MemoryBrightness       = 0.15
	MemorySaturation       = 0.3
	// How many tiles past its own tile a feature's sprite can reach, so that remembering a feature also bakes the
	// memory regions its sprite reaches into again
	MemoryFeatureReachTiles = 2
	// Regions that come into view beyond these are drawn black until they're baked, a few frames later, so that zooming
	// out doesn't stall a frame
	MaxMemoryBakesPerFrame = 4
	// How far off screen a memory region's image is kept before it's freed
	MemoryImageFreeMarginPx = 256
)
