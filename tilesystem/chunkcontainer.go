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
	chunks map[utils.CellKey]*chunk
	atlas  map[data.TileEnum]tileDef

	saveChunkCh  chan saveRequest
	loadReqCh    chan loadRequest
	loadResultCh chan loadResult
	saveResultCh chan saveResult

	chunkMeta map[utils.CellKey]*chunkLoadMeta

	currentTickEIdsInChunks map[utils.CellKey][]common.EntityId
	pendingEvictEntityIDs   map[utils.CellKey][]common.EntityId
}

func NewChunkContainer() *ChunkContainer {
	cc := &ChunkContainer{
		chunks: make(map[utils.CellKey]*chunk),

		saveChunkCh: make(chan saveRequest, 128),
		// Unbuffered on purpose: prevents stale load backlog when chunkloader moves quickly and remains deterministic.
		loadReqCh:    make(chan loadRequest),
		loadResultCh: make(chan loadResult, 1024),
		saveResultCh: make(chan saveResult, 128),

		chunkMeta:             make(map[utils.CellKey]*chunkLoadMeta),
		pendingEvictEntityIDs: make(map[utils.CellKey][]common.EntityId),
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
