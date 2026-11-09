package tilesystem

import (
	"ebittest/assetmanager"
	"ebittest/climate"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/json"
	"fmt"
	"os"
)

type ChunkContainer struct {
	// Active chunks, the only ones the simulation sees
	chunks       map[utils.Vec2i]*chunk
	atlas        map[data.TileEnum]tileDef
	featureAtlas map[data.FeatureEnum]featureDef
	blueprints   *ecs.BlueprintAtlas
	sightVersion uint64
	// For each size of body that's asked about, until a chunk comes or goes, or a tile or feature changes
	regionGraphs map[bodyBox]*regionGraph
	// Goes up whenever where bodies fit is forgotten, so that a lookup holding on to what it found knows it's gone
	bodyFitsVersion uint64
	// How far the light of the furthest-reaching feature goes
	maxLightRange int

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
	planet climate.Planet
	memory memory
	// The tiles something harvestable's been taken from since they were last looked at for exhausting the harvest area
	harvested []utils.Vec2i

	rooms map[uint64]*room
	// The chunks with pieces in rooms, active or not
	roomChunks map[utils.Vec2i]*roomChunk
	nextRoomId uint64
	roomsStale bool
	// Since the roof area was last put over newly enclosed tiles
	enclosed []enclosedTile
}

func NewChunkContainer() *ChunkContainer {
	cc := &ChunkContainer{
		chunks:     make(map[utils.Vec2i]*chunk),
		blueprints: ecs.NewBlueprintAtlas(),

		ioRequests: newIOQueue(),
		ioResults:  make(chan ioResult, 1024),

		resident:       make(map[utils.Vec2i]residentChunk),
		loadsInFlight:  make(map[utils.Vec2i]struct{}),
		loadFailures:   make(map[utils.Vec2i]int),
		evictedChunks:  make(map[utils.Vec2i]pendingChunk),
		queuedEntities: make(map[utils.Vec2i]map[common.EntityId]pendingEntity),
		memory:         newMemory(),
		rooms:          map[uint64]*room{},
		roomChunks:     map[utils.Vec2i]*roomChunk{},
	}

	cc.StartIOThread()

	return cc
}

// Drop tables are checked against the item atlas, so that one naming an item that doesn't exist fails here rather than
// when the tile is destroyed. Images come from the asset list.
func (cc *ChunkContainer) GenerateTileAtlasFromJson(input string, items *ecs.ItemAtlas, assets *assetmanager.AssetManager) error {
	f, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("error reading tile atlas json file: %w", err)
	}

	var tileDefs []tileDefDTO
	err = json.Unmarshal(f, &tileDefs)
	if err != nil {
		return fmt.Errorf("error unmarshalling tile atlas json: %w", err)
	}

	atlas, err := dtosToDefsMap(tileDefs, items, assets)
	if err != nil {
		return fmt.Errorf("error converting tile atlas DTOs to defs: %w", err)
	}

	cc.atlas = atlas

	return nil
}
