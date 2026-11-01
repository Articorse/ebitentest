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
	deletes []utils.Vec2i
	memory  map[utils.Vec2i]memoryRegionDto
}

type commitResult struct {
	committed bool
	err       error
}

type inFlightCommit struct {
	generation     uint64
	chunkVersions  map[utils.Vec2i]uint64
	entityVersions map[utils.Vec2i]map[common.EntityId]uint64
	deletes        []utils.Vec2i
	memoryVersions map[utils.Vec2i]uint64
}

func (cc *ChunkContainer) SetGeneration(generation uint64) {
	cc.generation = generation
}

// Must be set before the first chunk is requested, since chunks are generated on the I/O thread
func (cc *ChunkContainer) SetSeed(seed uint64) {
	cc.seed = seed
}

func (cc *ChunkContainer) Seed() uint64 {
	return cc.seed
}

func (cc *ChunkContainer) IsCommitting() bool {
	return cc.commitInFlight != nil
}

func (cc *ChunkContainer) StartCommit(header world.Header, ecsCont *ecs.ECSContainer) error {
	if cc.commitInFlight != nil {
		return fmt.Errorf("commit of generation %d still in flight", cc.commitInFlight.generation)
	}

	eIdsInChunks, err := cc.getEntityIdsInChunks(ecsCont)
	if err != nil {
		return fmt.Errorf("failed to get entity IDs in chunks: %w", err)
	}
	cc.currentTickEIdsInChunks = eIdsInChunks

	req := commitRequest{
		chunks:  make(map[utils.Vec2i]chunkDto, len(cc.chunks)+len(cc.evictedChunks)),
		appends: make(map[utils.Vec2i]map[common.EntityId][]ecs.ComponentDto, len(cc.queuedEntities)),
		memory:  make(map[utils.Vec2i]memoryRegionDto),
	}
	inFlight := &inFlightCommit{
		generation:     cc.generation + 1,
		chunkVersions:  make(map[utils.Vec2i]uint64, len(cc.evictedChunks)),
		entityVersions: make(map[utils.Vec2i]map[common.EntityId]uint64, len(cc.queuedEntities)),
		memoryVersions: make(map[utils.Vec2i]uint64),
	}

	for pos, r := range cc.memory.regions {
		if !r.dirty {
			continue
		}
		// Copied, since the main thread keeps remembering while the commit is written
		req.memory[pos] = memoryRegionDto{X: pos.X, Y: pos.Y, Tiles: append([]data.TileEnum(nil), r.tiles[:]...)}
		inFlight.memoryVersions[pos] = r.version
	}

	for pos, c := range cc.chunks {
		cDto, err := cc.chunkToDto(c, pos, ecsCont)
		if err != nil {
			return fmt.Errorf("failed to convert chunk at %v to DTO: %w", pos, err)
		}

		if cDto.needsSaving() {
			req.chunks[pos] = cDto
		} else {
			req.deletes = append(req.deletes, pos)
		}
	}

	// Maps are cloned, since the main thread keeps adding to them while the commit is written
	for pos, evicted := range cc.evictedChunks {
		inFlight.chunkVersions[pos] = evicted.version

		if !evicted.dto.needsSaving() {
			req.deletes = append(req.deletes, pos)
			continue
		}

		cDto := evicted.dto
		cDto.Entities = maps.Clone(evicted.dto.Entities)
		req.chunks[pos] = cDto
	}
	inFlight.deletes = req.deletes

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
	cc.ioRequests.push(ioRequest{commit: &req})

	return nil
}

func (cc *ChunkContainer) CommitAndWait(header world.Header, ecsCont *ecs.ECSContainer) error {
	for cc.commitInFlight != nil {
		cc.handleIOResult(<-cc.ioResults)
	}

	if err := cc.StartCommit(header, ecsCont); err != nil {
		return err
	}

	generation := cc.commitInFlight.generation
	for cc.commitInFlight != nil {
		cc.handleIOResult(<-cc.ioResults)
	}

	if cc.generation != generation {
		return fmt.Errorf("commit of generation %d failed", generation)
	}

	return nil
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

	// A region remembered more since the commit started still needs saving
	for pos, version := range inFlight.memoryVersions {
		if r, ok := cc.memory.regions[pos]; ok && r.version == version {
			r.dirty = false
		}
	}

	// Resident copies of these chunks were loaded from files this commit has since replaced
	for pos := range inFlight.chunkVersions {
		delete(cc.resident, pos)
	}
	for pos := range inFlight.entityVersions {
		delete(cc.resident, pos)
	}
	for _, pos := range inFlight.deletes {
		delete(cc.resident, pos)
	}

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

// Runs on the IO thread. Every chunk file is written under a temporary name first, and every deletion is marked,
// then the header is replaced, committing the save, and only then are the temporary files moved into place and the
// marked files deleted. RecoverChunkFiles finishes or discards that last step if the game stops in the middle of it.
func (cc *ChunkContainer) commit(req commitRequest) (committed bool, err error) {
	generation := req.header.Generation
	var tmpPaths []string
	var deleteMarkers []string

	discard := func() {
		for _, p := range tmpPaths {
			os.Remove(p)
		}
		for _, p := range deleteMarkers {
			os.Remove(p)
		}
	}

	writeTmp := func(pos utils.Vec2i, cDto chunkDto) error {
		p := chunkTmpFilePath(pos, generation)
		if err := writeGobFile(p, cDto); err != nil {
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

	for pos, mDto := range req.memory {
		p := memoryTmpFilePath(pos, generation)
		if err := writeGobFile(p, mDto); err != nil {
			discard()
			return false, fmt.Errorf("failed to write memory region at %v: %w", pos, err)
		}
		tmpPaths = append(tmpPaths, p)
	}

	for _, pos := range req.deletes {
		if _, err := os.Stat(chunkFilePath(pos)); errors.Is(err, os.ErrNotExist) {
			continue
		}

		p := chunkDeleteMarkerPath(pos, generation)
		if err := writeEmptyFile(p); err != nil {
			discard()
			return false, fmt.Errorf("failed to mark chunk at %v for deletion: %w", pos, err)
		}
		deleteMarkers = append(deleteMarkers, p)
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

	for _, p := range deleteMarkers {
		errs = errors.Join(errs, deleteMarkedFile(p, generation))
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
		if strings.HasSuffix(entry.Name(), ".deleted") {
			p := filepath.Join(data.ChunkDataDir, entry.Name())
			_, generationStr, found := cutLast(strings.TrimSuffix(entry.Name(), ".deleted"), ".")
			generation, parseErr := strconv.ParseUint(generationStr, 10, 64)

			if found && parseErr == nil && generation == committedGeneration {
				errs = errors.Join(errs, deleteMarkedFile(p, generation))
			} else if err := os.Remove(p); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to remove %s: %w", p, err))
			}
			continue
		}

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

func chunkFilePath(pos utils.Vec2i) string {
	return fmt.Sprintf(data.ChunkDataFilePathTemplate, pos.X, pos.Y)
}

func chunkDeleteMarkerPath(pos utils.Vec2i, generation uint64) string {
	return fmt.Sprintf(data.ChunkDataFilePathTemplate+".%d.deleted", pos.X, pos.Y, generation)
}

// The marker is only removed once the file it marks is gone, so that recovery can retry the deletion
func deleteMarkedFile(markerPath string, generation uint64) error {
	p := strings.TrimSuffix(markerPath, fmt.Sprintf(".%d.deleted", generation))
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to delete %s: %w", p, err)
	}

	if err := os.Remove(markerPath); err != nil {
		return fmt.Errorf("failed to remove %s: %w", markerPath, err)
	}

	return nil
}

func writeEmptyFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}

	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("error syncing file: %w", err)
	}

	return file.Close()
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

func writeGobFile(filePath string, v any) (err error) {
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

	if err := gob.NewEncoder(file).Encode(v); err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("error syncing file: %w", err)
	}

	return nil
}
