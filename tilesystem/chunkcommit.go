package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"ebittest/world"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type pendingChunk struct {
	dto     chunkDto
	version uint64
}

type pendingEntity struct {
	dtos    []ecs.ComponentDto
	version uint64
}

type commitRequest struct {
	header  world.Header
	chunks  map[utils.Vec2i]chunkDto
	appends map[utils.Vec2i]map[common.EntityId][]ecs.ComponentDto
}

type commitResult struct {
	committed bool
	err       error
}

type inFlightCommit struct {
	generation     uint64
	chunkVersions  map[utils.Vec2i]uint64
	entityVersions map[utils.Vec2i]map[common.EntityId]uint64
}

func (cc *ChunkContainer) SetGeneration(generation uint64) {
	cc.generation = generation
}

func (cc *ChunkContainer) IsCommitting() bool {
	return cc.commitInFlight != nil
}

func (cc *ChunkContainer) StartCommit(header world.Header, ecsCont *ecs.ECSContainer) error {
	if cc.commitInFlight != nil {
		return fmt.Errorf("commit of generation %d still in flight", cc.commitInFlight.generation)
	}

	eIdsInChunks, _, err := cc.getEntityIdsInChunks(ecsCont)
	if err != nil {
		return fmt.Errorf("failed to get entity IDs in chunks: %w", err)
	}
	cc.currentTickEIdsInChunks = eIdsInChunks

	req := commitRequest{
		chunks:  make(map[utils.Vec2i]chunkDto, len(cc.chunks)+len(cc.evictedChunks)),
		appends: make(map[utils.Vec2i]map[common.EntityId][]ecs.ComponentDto, len(cc.queuedEntities)),
	}
	inFlight := &inFlightCommit{
		generation:     cc.generation + 1,
		chunkVersions:  make(map[utils.Vec2i]uint64, len(cc.evictedChunks)),
		entityVersions: make(map[utils.Vec2i]map[common.EntityId]uint64, len(cc.queuedEntities)),
	}

	for pos, c := range cc.chunks {
		cDto, err := cc.chunkToDto(c, pos, ecsCont)
		if err != nil {
			return fmt.Errorf("failed to convert chunk at %v to DTO: %w", pos, err)
		}
		req.chunks[pos] = cDto
	}

	// Maps are cloned, since the main thread keeps adding to them while the commit is written
	for pos, evicted := range cc.evictedChunks {
		cDto := evicted.dto
		cDto.Entities = maps.Clone(evicted.dto.Entities)
		req.chunks[pos] = cDto
		inFlight.chunkVersions[pos] = evicted.version
	}

	for pos, ents := range cc.queuedEntities {
		req.appends[pos] = make(map[common.EntityId][]ecs.ComponentDto, len(ents))
		inFlight.entityVersions[pos] = make(map[common.EntityId]uint64, len(ents))
		for e, queued := range ents {
			req.appends[pos][e] = queued.dtos
			inFlight.entityVersions[pos][e] = queued.version
		}
	}

	header.Generation = inFlight.generation
	req.header = header

	cc.commitInFlight = inFlight
	cc.commitReqCh <- req

	return nil
}

func (cc *ChunkContainer) CommitAndWait(header world.Header, ecsCont *ecs.ECSContainer) error {
	if cc.commitInFlight != nil {
		cc.finishCommit(<-cc.commitResultCh)
	}

	if err := cc.StartCommit(header, ecsCont); err != nil {
		return err
	}

	res := <-cc.commitResultCh
	cc.finishCommit(res)

	return res.err
}

func (cc *ChunkContainer) drainCommitResults() {
	for {
		select {
		case res := <-cc.commitResultCh:
			cc.finishCommit(res)
		default:
			return
		}
	}
}

func (cc *ChunkContainer) finishCommit(res commitResult) {
	inFlight := cc.commitInFlight
	cc.commitInFlight = nil

	if res.err != nil {
		log.Printf("Error during commit of generation %d (committed: %v): %v", inFlight.generation, res.committed, res.err)
	}

	if !res.committed {
		return
	}

	cc.generation = inFlight.generation

	// Entries replaced since the commit started hold newer data, so only the committed versions are dropped
	for pos, version := range inFlight.chunkVersions {
		if evicted, ok := cc.evictedChunks[pos]; ok && evicted.version == version {
			delete(cc.evictedChunks, pos)
		}
	}

	for pos, ents := range inFlight.entityVersions {
		for e, version := range ents {
			if queued, ok := cc.queuedEntities[pos][e]; ok && queued.version == version {
				delete(cc.queuedEntities[pos], e)
			}
		}

		if len(cc.queuedEntities[pos]) == 0 {
			delete(cc.queuedEntities, pos)
		}
	}
}

func (cc *ChunkContainer) queueEntity(pos utils.Vec2i, e common.EntityId, dtos []ecs.ComponentDto) {
	cc.pendingVersion++

	if evicted, ok := cc.evictedChunks[pos]; ok {
		if evicted.dto.Entities == nil {
			evicted.dto.Entities = make(map[common.EntityId][]ecs.ComponentDto)
		}
		evicted.dto.Entities[e] = dtos
		evicted.version = cc.pendingVersion
		cc.evictedChunks[pos] = evicted
		return
	}

	if _, ok := cc.queuedEntities[pos]; !ok {
		cc.queuedEntities[pos] = make(map[common.EntityId]pendingEntity)
	}
	cc.queuedEntities[pos][e] = pendingEntity{dtos: dtos, version: cc.pendingVersion}
}

// Runs on the IO thread. Every chunk file is written under a temporary name first, then the header is replaced,
// committing the save, and only then are the temporary files moved into place. RecoverChunkFiles finishes or discards
// that last step if the game stops in the middle of it.
func (cc *ChunkContainer) commit(req commitRequest) (committed bool, err error) {
	generation := req.header.Generation
	var tmpPaths []string

	discard := func() {
		for _, p := range tmpPaths {
			os.Remove(p)
		}
	}

	writeTmp := func(pos utils.Vec2i, cDto chunkDto) error {
		p := chunkTmpFilePath(pos, generation)
		if err := writeChunkFile(p, cDto); err != nil {
			return err
		}
		tmpPaths = append(tmpPaths, p)
		return nil
	}

	for pos, cDto := range req.chunks {
		if err := writeTmp(pos, cDto); err != nil {
			discard()
			return false, fmt.Errorf("failed to write chunk at %v: %w", pos, err)
		}
	}

	for pos, ents := range req.appends {
		cDto, exists, err := loadChunkGob(pos)
		if err != nil {
			discard()
			return false, fmt.Errorf("failed to load chunk at %v to append entities to: %w", pos, err)
		}

		if !exists {
			c, err := cc.newChunkData(pos)
			if err != nil {
				discard()
				return false, fmt.Errorf("failed to create chunk at %v to append entities to: %w", pos, err)
			}
			cDto = chunkDto{X: pos.X, Y: pos.Y, Tiles: c.tiles}
		}

		if cDto.Entities == nil {
			cDto.Entities = make(map[common.EntityId][]ecs.ComponentDto, len(ents))
		}
		maps.Copy(cDto.Entities, ents)

		if err := writeTmp(pos, cDto); err != nil {
			discard()
			return false, fmt.Errorf("failed to write chunk at %v: %w", pos, err)
		}
	}

	if err := world.WriteHeader(req.header); err != nil {
		discard()
		return false, fmt.Errorf("failed to write world header: %w", err)
	}

	var errs error
	for _, p := range tmpPaths {
		if err := os.Rename(p, committedFilePath(p, generation)); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to move %s into place: %w", p, err))
		}
	}

	return true, errs
}

func RecoverChunkFiles(committedGeneration uint64) error {
	entries, err := os.ReadDir(data.ChunkDataDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("error reading chunk data directory: %w", err)
	}

	var errs error
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}

		p := filepath.Join(data.ChunkDataDir, entry.Name())
		base, generationStr, found := cutLast(strings.TrimSuffix(entry.Name(), ".tmp"), ".")
		generation, parseErr := strconv.ParseUint(generationStr, 10, 64)

		if found && parseErr == nil && generation == committedGeneration {
			if err := os.Rename(p, filepath.Join(data.ChunkDataDir, base)); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to move %s into place: %w", p, err))
			}
			continue
		}

		if err := os.Remove(p); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to remove %s: %w", p, err))
		}
	}

	return errs
}

func chunkTmpFilePath(pos utils.Vec2i, generation uint64) string {
	return fmt.Sprintf(data.ChunkDataFilePathTemplate+".%d.tmp", pos.X, pos.Y, generation)
}

func committedFilePath(tmpPath string, generation uint64) string {
	return strings.TrimSuffix(tmpPath, fmt.Sprintf(".%d.tmp", generation))
}

func cutLast(s string, sep string) (before string, after string, found bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}

func writeChunkFile(filePath string, cDto chunkDto) (err error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close file: %w", cerr))
		}
	}()

	if err := gob.NewEncoder(file).Encode(cDto); err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("error syncing file: %w", err)
	}

	return nil
}
