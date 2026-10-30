package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/json"
	"fmt"
	"os"
)

type ChunkContainer struct {
	chunks map[utils.Vec2i]*chunk
	atlas  map[data.TileEnum]tileDef

	loadReqCh      chan loadRequest
	loadResultCh   chan loadResult
	commitReqCh    chan commitRequest
	commitResultCh chan commitResult

	chunkMeta map[utils.Vec2i]*chunkLoadMeta

	currentTickEIdsInChunks map[utils.Vec2i][]common.EntityId

	// Chunks and entities that left the simulation are kept here until a commit writes them to disk,
	// so that the files on disk always match the world as of the last commit.
	evictedChunks  map[utils.Vec2i]pendingChunk
	queuedEntities map[utils.Vec2i]map[common.EntityId]pendingEntity
	pendingVersion uint64

	generation     uint64
	commitInFlight *inFlightCommit
}

func NewChunkContainer() *ChunkContainer {
	cc := &ChunkContainer{
		chunks: make(map[utils.Vec2i]*chunk),

		// Unbuffered on purpose: prevents stale load backlog when chunkloader moves quickly and remains deterministic.
		loadReqCh:    make(chan loadRequest),
		loadResultCh: make(chan loadResult, 1024),
		// Only one commit is ever in flight, so neither side can block
		commitReqCh:    make(chan commitRequest, 1),
		commitResultCh: make(chan commitResult, 1),

		chunkMeta:      make(map[utils.Vec2i]*chunkLoadMeta),
		evictedChunks:  make(map[utils.Vec2i]pendingChunk),
		queuedEntities: make(map[utils.Vec2i]map[common.EntityId]pendingEntity),
	}

	cc.StartIOThreads()

	return cc
}

func (cc *ChunkContainer) GenerateTileAtlasFromJson(input string) error {
	f, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("error reading tile atlas json file: %w", err)
	}

	var tileDefs []tileDefDTO
	err = json.Unmarshal(f, &tileDefs)
	if err != nil {
		return fmt.Errorf("error unmarshalling tile atlas json: %w", err)
	}

	atlas, err := dtosToDefsMap(tileDefs)
	if err != nil {
		return fmt.Errorf("error converting tile atlas DTOs to defs: %w", err)
	}

	cc.atlas = atlas

	return nil
}
