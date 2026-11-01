package world

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"encoding/gob"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

type Header struct {
	FormatVersion int
	Generation    uint64
	NextEntity    common.EntityId
	TickIdx       uint64
	PlayerEntity  common.EntityId
	RngState      []byte
	Entities      map[common.EntityId][]ecs.ComponentDto
}

// Chunkloaders and their compositions are stored here rather than in chunks,
// since the chunks they would be stored in can't be loaded without them.
func Snapshot(playerEntity common.EntityId, ecsCont *ecs.ECSContainer) (Header, error) {
	rngState, err := ecsCont.RngState()
	if err != nil {
		return Header{}, fmt.Errorf("error getting RNG state: %w", err)
	}

	h := Header{
		FormatVersion: data.WorldFormatVersion,
		NextEntity:    ecsCont.NextEntity(),
		TickIdx:       ecsCont.TickIdx,
		PlayerEntity:  playerEntity,
		RngState:      rngState,
		Entities:      make(map[common.EntityId][]ecs.ComponentDto),
	}

	for _, loader := range ecsCont.ChunkLoaders.GetEntities() {
		members, err := ecsCont.GetCompositionComponents(loader)
		if err != nil {
			return Header{}, fmt.Errorf("error getting composition of chunkloader entity %d: %w", loader, err)
		}
		maps.Copy(h.Entities, members)
	}

	return h, nil
}

func Load(ecsCont *ecs.ECSContainer) (h Header, exists bool, err error) {
	h, exists, err = readHeader()
	if err != nil || !exists {
		return h, exists, err
	}

	if h.FormatVersion != data.WorldFormatVersion {
		return h, true, fmt.Errorf(
			"world was saved in format version %d, but this build only reads version %d; delete %s to start a new world",
			h.FormatVersion, data.WorldFormatVersion, data.ChunkDataDir,
		)
	}

	ecsCont.SetNextEntity(h.NextEntity)
	ecsCont.TickIdx = h.TickIdx

	// Headers saved before RNG state was stored have none, in which case the seeded state is kept
	if len(h.RngState) > 0 {
		if err := ecsCont.SetRngState(h.RngState); err != nil {
			return h, true, fmt.Errorf("error restoring RNG state: %w", err)
		}
	}

	// Insertion order decides the order systems iterate over entities, so it must not depend on map order
	for _, e := range slices.Sorted(maps.Keys(h.Entities)) {
		compDtos := h.Entities[e]
		comps := make([]ecs.Component, 0, len(compDtos))
		for _, compDto := range compDtos {
			comp, err := ecs.DtoToComponent(compDto)
			if err != nil {
				return h, true, fmt.Errorf("error converting component of entity %d: %w", e, err)
			}
			comps = append(comps, comp)
		}

		if err := ecsCont.AddEntityWithId(e, comps...); err != nil {
			return h, true, fmt.Errorf("error adding entity %d: %w", e, err)
		}
	}

	return h, true, nil
}

func Reset() error {
	if err := os.RemoveAll(data.ChunkDataDir); err != nil {
		return fmt.Errorf("error removing chunk data directory: %w", err)
	}

	return nil
}

// Replacing the header is the point at which a world save is committed
func WriteHeader(h Header) error {
	dir := filepath.Dir(data.WorldHeaderFilePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	file, err := os.CreateTemp(dir, "world-*.tmp")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}

	if err := gob.NewEncoder(file).Encode(h); err != nil {
		file.Close()
		os.Remove(file.Name())
		return fmt.Errorf("error encoding world header: %w", err)
	}

	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(file.Name())
		return fmt.Errorf("error syncing temporary file: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(file.Name())
		return fmt.Errorf("error closing temporary file: %w", err)
	}

	if err := os.Rename(file.Name(), data.WorldHeaderFilePath); err != nil {
		os.Remove(file.Name())
		return fmt.Errorf("error replacing world header: %w", err)
	}

	return nil
}

func readHeader() (h Header, exists bool, err error) {
	file, err := os.Open(data.WorldHeaderFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Header{}, false, nil
		}
		return Header{}, false, fmt.Errorf("error opening world header: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close file: %w", cerr))
		}
	}()

	if err := gob.NewDecoder(file).Decode(&h); err != nil {
		return Header{}, true, fmt.Errorf("error decoding world header: %w", err)
	}

	return h, true, nil
}
