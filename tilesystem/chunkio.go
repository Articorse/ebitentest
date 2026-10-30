package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

// Exactly one of the fields is set
type ioRequest struct {
	load   *utils.Vec2i
	commit *commitRequest
}

// Exactly one of the fields is set
type ioResult struct {
	load   *loadResult
	commit *commitResult
}

type loadResult struct {
	pos   utils.Vec2i
	chunk *chunk
	ents  map[common.EntityId][]ecs.ComponentDto
	err   error
}

type residentChunk struct {
	chunk *chunk
	ents  map[common.EntityId][]ecs.ComponentDto
}

// Unbounded, so that the main thread never blocks when queueing a request
type ioQueue struct {
	mtx      sync.Mutex
	cond     *sync.Cond
	requests []ioRequest
}

func newIOQueue() *ioQueue {
	q := &ioQueue{}
	q.cond = sync.NewCond(&q.mtx)
	return q
}

func (q *ioQueue) push(req ioRequest) {
	q.mtx.Lock()
	q.requests = append(q.requests, req)
	q.mtx.Unlock()
	q.cond.Signal()
}

func (q *ioQueue) pop() ioRequest {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	for len(q.requests) == 0 {
		q.cond.Wait()
	}

	req := q.requests[0]
	q.requests = q.requests[1:]
	return req
}

// The IO thread is the only owner of the chunk files. Requests are processed in the order they were queued,
// so a load always sees the files of every commit queued before it.
func (cc *ChunkContainer) StartIOThread() {
	go func() {
		for {
			req := cc.ioRequests.pop()

			switch {
			case req.load != nil:
				res := cc.load(*req.load)
				cc.ioResults <- ioResult{load: &res}
			case req.commit != nil:
				committed, err := cc.commit(*req.commit)
				cc.ioResults <- ioResult{commit: &commitResult{committed: committed, err: err}}
			}
		}
	}()
}

func (cc *ChunkContainer) load(pos utils.Vec2i) loadResult {
	cDto, exists, err := loadChunkGob(pos)
	if err != nil {
		return loadResult{pos: pos, err: err}
	}

	if !exists {
		c, err := cc.newChunkData(pos)
		if err != nil {
			return loadResult{pos: pos, err: fmt.Errorf("failed to create new chunk: %w", err)}
		}
		return loadResult{pos: pos, chunk: c}
	}

	return loadResult{pos: pos, chunk: cc.dtoToChunkData(cDto), ents: cDto.Entities}
}

func (cc *ChunkContainer) requestLoad(pos utils.Vec2i) {
	if _, ok := cc.loadsInFlight[pos]; ok {
		return
	}

	cc.loadsInFlight[pos] = struct{}{}
	cc.ioRequests.push(ioRequest{load: &pos})
}

func (cc *ChunkContainer) drainIOResults() {
	for {
		select {
		case res := <-cc.ioResults:
			cc.handleIOResult(res)
		default:
			return
		}
	}
}

func (cc *ChunkContainer) handleIOResult(res ioResult) {
	switch {
	case res.load != nil:
		cc.handleLoadResult(*res.load)
	case res.commit != nil:
		cc.finishCommit(*res.commit)
	}
}

func (cc *ChunkContainer) handleLoadResult(res loadResult) {
	delete(cc.loadsInFlight, res.pos)

	if res.err != nil {
		cc.loadFailures[res.pos]++
		log.Printf("Failed to load chunk at %v (attempt %d): %v", res.pos, cc.loadFailures[res.pos], res.err)

		if cc.loadFailures[res.pos] < data.ChunkReloadRetries {
			cc.requestLoad(res.pos)
		}
		return
	}
	delete(cc.loadFailures, res.pos)

	// Active and evicted chunks hold newer data than the file this was loaded from
	if _, ok := cc.chunks[res.pos]; ok {
		return
	}
	if _, ok := cc.evictedChunks[res.pos]; ok {
		return
	}

	cc.resident[res.pos] = residentChunk{chunk: res.chunk, ents: res.ents}
}

func (cc *ChunkContainer) isAvailable(pos utils.Vec2i) bool {
	if _, ok := cc.resident[pos]; ok {
		return true
	}
	_, ok := cc.evictedChunks[pos]
	return ok
}

// Blocks until the chunk is resident, which only happens when a loader outruns the prefetch margin
func (cc *ChunkContainer) waitUntilAvailable(pos utils.Vec2i) error {
	delete(cc.loadFailures, pos)

	for !cc.isAvailable(pos) {
		if _, ok := cc.loadsInFlight[pos]; !ok {
			if cc.loadFailures[pos] >= data.ChunkReloadRetries {
				return fmt.Errorf("failed to load chunk at %v after %d attempts", pos, cc.loadFailures[pos])
			}
			cc.requestLoad(pos)
		}

		cc.handleIOResult(<-cc.ioResults)
	}

	return nil
}

func loadChunkGob(cellKey utils.Vec2i) (cDto chunkDto, exists bool, err error) {
	file, err := os.Open(fmt.Sprintf(data.ChunkDataFilePathTemplate, cellKey.X, cellKey.Y))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return chunkDto{}, false, nil
		}
		return chunkDto{}, false, fmt.Errorf("error opening file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close file: %w", cerr))
		}
	}()

	var data chunkDto
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return chunkDto{}, true, fmt.Errorf("error decoding data: %w", err)
	}

	return data, true, nil
}

func postLoadChunk(c *chunk, atlas map[data.TileEnum]tileDef) error {
	for _, tEnum := range c.tiles {
		if _, ok := atlas[tEnum]; !ok {
			return fmt.Errorf("tile enum %d not found in atlas", tEnum)
		}
	}

	return nil
}
