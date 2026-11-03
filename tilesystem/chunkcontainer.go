package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

type ChunkContainer struct {
	// Active chunks, the only ones the simulation sees
	chunks       map[utils.Vec2i]*chunk
	atlas        map[data.TileEnum]tileDef
	featureAtlas map[data.FeatureEnum]featureDef

	ioRequests *ioQueue
	ioResults  chan ioResult

	resident      map[utils.Vec2i]residentChunk
	loadsInFlight map[utils.Vec2i]struct{}
	loadFailures  map[utils.Vec2i]int

	currentTickEIdsInChunks map[utils.Vec2i][]common.EntityId

	// Chunks and entities that left the simulation are kept here until a commit writes them to disk,
	// so that the files on disk always match the world as of the last commit.
	evictedChunks  map[utils.Vec2i]pendingChunk
	queuedEntities map[utils.Vec2i]map[common.EntityId]pendingEntity
	pendingVersion uint64

	generation     uint64
	commitInFlight *inFlightCommit

	seed   uint64
	memory memory
	// Reused for every region baked, since each bake writes it before drawing it
	memoryRoofShade *ebiten.Image
}

func NewChunkContainer() *ChunkContainer {
	cc := &ChunkContainer{
		chunks: make(map[utils.Vec2i]*chunk),

		ioRequests: newIOQueue(),
		ioResults:  make(chan ioResult, 1024),

		resident:       make(map[utils.Vec2i]residentChunk),
		loadsInFlight:  make(map[utils.Vec2i]struct{}),
		loadFailures:   make(map[utils.Vec2i]int),
		evictedChunks:  make(map[utils.Vec2i]pendingChunk),
		queuedEntities: make(map[utils.Vec2i]map[common.EntityId]pendingEntity),
		memory:         newMemory(),
	}

	cc.StartIOThread()

	return cc
}

// Drop tables are checked against the item atlas, so that one naming an item that doesn't exist fails here rather than
// when the tile is destroyed
func (cc *ChunkContainer) GenerateTileAtlasFromJson(input string, items *ecs.ItemAtlas) error {
	f, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("error reading tile atlas json file: %w", err)
	}

	var tileDefs []tileDefDTO
	err = json.Unmarshal(f, &tileDefs)
	if err != nil {
		return fmt.Errorf("error unmarshalling tile atlas json: %w", err)
	}

	atlas, err := dtosToDefsMap(tileDefs, items)
	if err != nil {
		return fmt.Errorf("error converting tile atlas DTOs to defs: %w", err)
	}

	cc.atlas = atlas

	return nil
}
