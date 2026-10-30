package world

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type header struct {
	NextEntity   common.EntityId
	TickIdx      uint64
	PlayerEntity common.EntityId
	Entities     map[common.EntityId][]ecs.ComponentDto
}

// Chunkloaders and their hierarchies are stored here rather than in chunks,
// since the chunks they would be stored in can't be loaded without them.
func Save(playerEntity common.EntityId, ecsCont *ecs.ECSContainer) error {
	hm := ecsCont.HierarchyManager

	h := header{
		NextEntity:   ecsCont.NextEntity(),
		TickIdx:      ecsCont.TickIdx,
		PlayerEntity: playerEntity,
		Entities:     make(map[common.EntityId][]ecs.ComponentDto),
	}

	for _, loader := range ecsCont.ChunkLoaders.GetEntities() {
		root := hm.GetRoot(loader, ecsCont)

		children, err := hm.GetChildrenDFS(root, ecsCont)
		if err != nil {
			return fmt.Errorf("error getting children of chunkloader entity %d: %w", root, err)
		}

		for _, e := range append([]common.EntityId{root}, children...) {
			h.Entities[e] = ecsCont.GetEntityComponents(e)
		}
	}

	return writeHeader(h)
}

func Load(ecsCont *ecs.ECSContainer) (playerEntity common.EntityId, exists bool, err error) {
	h, exists, err := readHeader()
	if err != nil || !exists {
		return -1, exists, err
	}

	ecsCont.SetNextEntity(h.NextEntity)
	ecsCont.TickIdx = h.TickIdx

	for e, compDtos := range h.Entities {
		comps := make([]ecs.Component, 0, len(compDtos))
		for _, compDto := range compDtos {
			comp, err := ecs.DtoToComponent(compDto)
			if err != nil {
				return -1, true, fmt.Errorf("error converting component of entity %d: %w", e, err)
			}
			comps = append(comps, comp)
		}

		if err := ecsCont.AddEntityWithId(e, comps...); err != nil {
			return -1, true, fmt.Errorf("error adding entity %d: %w", e, err)
		}
	}

	return h.PlayerEntity, true, nil
}

func Reset() error {
	if err := os.RemoveAll(data.ChunkDataDir); err != nil {
		return fmt.Errorf("error removing chunk data directory: %w", err)
	}

	return nil
}

func writeHeader(h header) error {
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

func readHeader() (h header, exists bool, err error) {
	file, err := os.Open(data.WorldHeaderFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return header{}, false, nil
		}
		return header{}, false, fmt.Errorf("error opening world header: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close file: %w", cerr))
		}
	}()

	if err := gob.NewDecoder(file).Decode(&h); err != nil {
		return header{}, true, fmt.Errorf("error decoding world header: %w", err)
	}

	return h, true, nil
}
