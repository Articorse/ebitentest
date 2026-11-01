package world_test

import (
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/world"
	"encoding/gob"
	"os"
	"strings"
	"testing"
)

func TestNoWorldToLoad(t *testing.T) {
	testutil.TempWorld(t)

	_, exists, err := world.Load(testutil.NewECS(t))
	if exists || err != nil {
		t.Fatalf("world exists %v, error %v", exists, err)
	}
}

func TestWorldInOtherFormatRefusedBeforeDecoding(t *testing.T) {
	testutil.TempWorld(t)
	if err := os.MkdirAll(data.ChunkDataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	file, err := os.Create(data.WorldHeaderFilePath)
	if err != nil {
		t.Fatal(err)
	}
	enc := gob.NewEncoder(file)
	if err := enc.Encode(data.WorldFormatVersion + 1); err != nil {
		t.Fatal(err)
	}
	if err := enc.Encode("a header this build can't decode"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	_, exists, err := world.Load(testutil.NewECS(t))
	if !exists || err == nil {
		t.Fatalf("world exists %v, error %v", exists, err)
	}
	if !strings.Contains(err.Error(), "delete "+data.ChunkDataDir) {
		t.Fatalf("error doesn't say to delete %s: %v", data.ChunkDataDir, err)
	}
}
