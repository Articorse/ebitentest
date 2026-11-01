package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"ebittest/world"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func chunkFileName(pos utils.Vec2f) string {
	c := utils.WorldPosToChunkGridPos(pos)
	return filepath.Base(fmt.Sprintf(data.ChunkDataFilePathTemplate, c.X, c.Y))
}

func assertChunkFiles(t *testing.T, want ...string) {
	t.Helper()

	if got := testutil.ChunkFiles(t); !slices.Equal(got, want) {
		t.Fatalf("chunk files %v, want %v", got, want)
	}
}

func TestUntouchedChunksAreNotSaved(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	s.MovePlayer(utils.Vec2f{X: 600, Y: 300})
	s.Tick(5)
	s.Commit()

	assertChunkFiles(t)
}

func TestChunkSavedWhileHoldingPersistentEntity(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	rock := testutil.AddMovable(t, s.ECS, nearPlayer, true)
	s.Tick(3)
	s.Commit()
	assertChunkFiles(t, chunkFileName(nearPlayer))

	s.ECS.ScheduleRemoveEntity(rock)
	s.Tick(3)
	s.Commit()
	assertChunkFiles(t)
}

func TestEvictedChunkFileDeletedAndRegenerated(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	x := s.ECS
	rock := testutil.AddMovable(t, x, nearPlayer, true)
	s.Tick(3)
	s.Commit()

	x.ScheduleRemoveEntity(rock)
	s.Tick(2)
	s.MovePlayer(elsewhere)
	s.Tick(5)
	assertChunkFiles(t, chunkFileName(nearPlayer))

	s.Commit()
	assertChunkFiles(t)

	s.MovePlayer(testutil.PlayerStart)
	s.Tick(5)
	if x.HasEntity(rock) {
		t.Fatal("removed rock came back")
	}
}

func TestEntityMovedIntoUnsavedChunkIsSaved(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	e := testutil.AddMovable(t, s.ECS, nearPlayer, true)
	testutil.SetPos(t, s.ECS, e, farAway)
	s.Tick(3)
	s.Commit()

	assertChunkFiles(t, chunkFileName(farAway))
}

func TestResumedWorldMatchesSavedOne(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	x := s.ECS
	rock := testutil.AddMovable(t, x, nearPlayer, true)
	item := testutil.AddMovable(t, x, testutil.PlayerStart, true)
	testutil.Attach(t, x, item, s.Player, ecs.Link_Held)
	s.Tick(30)
	s.Commit()

	r := testutil.StartSession(t)
	y := r.ECS
	if !r.Resumed || r.Player != s.Player {
		t.Fatalf("resumed %v with player %d, want player %d", r.Resumed, r.Player, s.Player)
	}
	if y.NextEntity() != x.NextEntity() {
		t.Fatalf("next entity %d, want %d", y.NextEntity(), x.NextEntity())
	}
	if !y.HasEntity(rock) {
		t.Fatal("rock wasn't restored")
	}
	if y.Parent(item) != r.Player || y.LinkKind(item) != ecs.Link_Held {
		t.Fatal("held item isn't held by the player any more")
	}
	if err := y.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestPeriodicCommitResumesTickAndRng(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	x := s.ECS
	x.Rng.IntN(1000)

	h, err := world.Snapshot(s.Chunks.Seed(), s.Player, x)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Chunks.StartCommit(h, x); err != nil {
		t.Fatal(err)
	}
	tick := x.TickIdx
	for s.Chunks.IsCommitting() {
		s.Tick(1)
	}
	want := x.Rng.IntN(1 << 30)

	y := testutil.NewECS(t)
	loaded, exists, err := world.Load(y)
	if err != nil || !exists {
		t.Fatalf("world exists %v, error %v", exists, err)
	}
	if loaded.TickIdx != tick {
		t.Fatalf("resumed at tick %d, want %d", loaded.TickIdx, tick)
	}
	if got := y.Rng.IntN(1 << 30); got != want {
		t.Fatal("RNG doesn't continue where the save left off")
	}
}

func TestRecoveryFinishesCommittedAndDiscardsUncommittedFiles(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	s.Commit()
	h, _, err := world.Load(testutil.NewECS(t))
	if err != nil {
		t.Fatal(err)
	}
	gen := h.Generation
	path := func(x, y int) string { return fmt.Sprintf(data.ChunkDataFilePathTemplate, x, y) }
	write := func(p string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(p), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(fmt.Sprintf("%s.%d.tmp", path(1, 1), gen))
	write(fmt.Sprintf("%s.%d.tmp", path(2, 2), gen+1))
	write(path(3, 3))
	write(fmt.Sprintf("%s.%d.deleted", path(3, 3), gen))
	write(path(4, 4))
	write(fmt.Sprintf("%s.%d.deleted", path(4, 4), gen+1))
	write(filepath.Join(data.ChunkDataDir, "world-123.tmp"))

	if err := tilesystem.RecoverChunkFiles(gen); err != nil {
		t.Fatal(err)
	}

	assertChunkFiles(t, filepath.Base(path(1, 1)), filepath.Base(path(4, 4)))
	if got, _ := os.ReadFile(path(1, 1)); string(got) != fmt.Sprintf("%s.%d.tmp", path(1, 1), gen) {
		t.Fatalf("committed temporary file wasn't moved into place, file holds %q", got)
	}
	entries, err := os.ReadDir(data.ChunkDataDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" || filepath.Ext(e.Name()) == ".deleted" {
			t.Errorf("%s left behind", e.Name())
		}
	}
}

func TestNewWorldIsDefinedByItsSeed(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)

	fresh := testutil.NewECS(t)
	fresh.SeedRng(testutil.WorldSeed)
	if s.ECS.Rng.Uint64() != fresh.Rng.Uint64() {
		t.Fatal("the new world's RNG isn't seeded from the world seed")
	}

	s.Commit()

	r := testutil.StartSession(t)
	if !r.Resumed || r.Chunks.Seed() != testutil.WorldSeed {
		t.Fatalf("resumed %v with seed %d, want the saved seed %d", r.Resumed, r.Chunks.Seed(), testutil.WorldSeed)
	}
}
