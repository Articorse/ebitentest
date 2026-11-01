package tilesystem_test

import (
	"bytes"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
)

var (
	nearPlayer = utils.Vec2f{X: 150, Y: 60}
	farAway    = utils.Vec2f{X: 2000, Y: 2000}
	elsewhere  = utils.Vec2f{X: 5000, Y: 5000}
)

func TestEntitiesLeavingActiveChunksPersistOrDespawn(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	x := s.ECS
	keep := testutil.AddMovable(t, x, nearPlayer, true)
	gone := testutil.AddMovable(t, x, nearPlayer, false)
	testutil.SetPos(t, x, keep, farAway)
	testutil.SetPos(t, x, gone, farAway)
	s.Tick(2)

	if x.HasEntity(keep) || x.HasEntity(gone) {
		t.Fatal("entities outside the active chunks are still simulated")
	}

	s.MovePlayer(farAway)
	s.Tick(5)

	if !x.HasEntity(keep) {
		t.Fatal("persistent entity didn't come back with its chunk")
	}
	if got := testutil.WorldPos(t, x, keep); got != farAway {
		t.Fatalf("persistent entity came back at %v, want %v", got, farAway)
	}
	if x.HasEntity(gone) {
		t.Fatal("non-persistent entity came back")
	}
}

func TestDespawnOfDriftingEntityLogged(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	e := testutil.AddMovable(t, s.ECS, nearPlayer, false)
	s.Tick(1)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	testutil.SetPos(t, s.ECS, e, elsewhere)
	s.Tick(1)

	if want := fmt.Sprintf("Despawning entity %d", e); !strings.Contains(buf.String(), want) {
		t.Fatalf("log %q doesn't mention %q", buf.String(), want)
	}
}

func TestCompositionStoredAndRestoredAsUnit(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	x := s.ECS
	crab := testutil.AddMovable(t, x, nearPlayer, true)
	claw := testutil.AddMovable(t, x, nearPlayer, true)
	fire := testutil.AddMovable(t, x, nearPlayer, false)
	item := testutil.AddMovable(t, x, nearPlayer, true)
	platform := testutil.AddMovable(t, x, nearPlayer.Add(utils.Vec2f{X: 20}), false)
	rider := testutil.AddMovable(t, x, nearPlayer.Add(utils.Vec2f{X: 20}), true)
	testutil.Attach(t, x, claw, crab, ecs.Link_Part)
	testutil.Attach(t, x, fire, crab, ecs.Link_Part)
	testutil.Attach(t, x, item, claw, ecs.Link_Held)
	testutil.Attach(t, x, rider, platform, ecs.Link_Attached)
	testutil.SetPos(t, x, claw, utils.Vec2f{X: 5})
	clawPos := testutil.WorldPos(t, x, claw)
	riderPos := testutil.WorldPos(t, x, rider)
	s.Tick(2)

	s.MovePlayer(elsewhere)
	s.Tick(5)
	for _, e := range []common.EntityId{crab, claw, fire, item, platform, rider} {
		if x.HasEntity(e) {
			t.Fatalf("entity %d still simulated after its chunk was deactivated", e)
		}
	}
	s.Commit()

	s.MovePlayer(testutil.PlayerStart)
	s.Tick(5)

	if !x.HasEntity(crab) || !x.HasEntity(claw) || !x.HasEntity(item) {
		t.Fatalf("crab %v, claw %v, held item %v came back", x.HasEntity(crab), x.HasEntity(claw), x.HasEntity(item))
	}
	if x.HasEntity(fire) || x.HasEntity(platform) {
		t.Fatalf("non-persistent fire %v or platform %v came back", x.HasEntity(fire), x.HasEntity(platform))
	}
	if x.Parent(claw) != crab || x.LinkKind(claw) != ecs.Link_Part {
		t.Fatalf("claw linked to %d as %d, want part of %d", x.Parent(claw), x.LinkKind(claw), crab)
	}
	if x.Parent(item) != claw || x.LinkKind(item) != ecs.Link_Held {
		t.Fatalf("item linked to %d as %d, want held by %d", x.Parent(item), x.LinkKind(item), claw)
	}
	if got := testutil.WorldPos(t, x, claw); got != clawPos {
		t.Fatalf("claw at %v, want %v", got, clawPos)
	}
	if !x.HasEntity(rider) || x.Parent(rider) != -1 {
		t.Fatalf("rider alive %v with parent %d, want stored on its own", x.HasEntity(rider), x.Parent(rider))
	}
	if got := testutil.WorldPos(t, x, rider); got != riderPos {
		t.Fatalf("rider at %v, want %v", got, riderPos)
	}
}

func TestTetheredEntityKeptInAnchorArea(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	x := s.ECS
	colonist := testutil.AddMovable(t, x, testutil.PlayerStart.Add(utils.Vec2f{X: 20}), true, ecs.NewTetherComponent(s.Player))
	vel, ok := x.Velocities.Get(colonist)
	if !ok {
		t.Fatal("colonist has no velocity")
	}
	vel.SetLocalVector(utils.Vec2f{X: 3})
	s.Tick(2)

	s.MovePlayer(utils.Vec2f{X: 1000, Y: 100})
	s.Tick(3)

	if !x.HasEntity(colonist) {
		t.Fatal("tethered colonist was unloaded")
	}
	colonistChunk := utils.WorldPosToChunkGridPos(testutil.WorldPos(t, x, colonist))
	playerChunk := utils.WorldPosToChunkGridPos(testutil.WorldPos(t, x, s.Player))
	if abs(colonistChunk.X-playerChunk.X) > testutil.PlayerLoaderRadius || abs(colonistChunk.Y-playerChunk.Y) > testutil.PlayerLoaderRadius {
		t.Fatalf("colonist in chunk %v, outside the active chunks around %v", colonistChunk, playerChunk)
	}
	if vel, _ = x.Velocities.Get(colonist); vel.LocalVector() != (utils.Vec2f{}) {
		t.Fatalf("relocated colonist kept moving at %v", vel.LocalVector())
	}
}

func TestEntityTetheredToMissingAnchorTreatedLikeAnyOther(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	loner := testutil.AddMovable(t, s.ECS, nearPlayer, false, ecs.NewTetherComponent(9999))
	s.Tick(1)

	s.MovePlayer(elsewhere)
	s.Tick(3)

	if s.ECS.HasEntity(loner) {
		t.Fatal("non-persistent entity tethered to a missing anchor wasn't despawned")
	}
}

func TestEvictedChunkRestoredFromMemoryBeforeCommit(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	x := s.ECS
	s.Commit()
	rock := testutil.AddMovable(t, x, nearPlayer, true)
	s.Tick(2)

	s.MovePlayer(elsewhere)
	s.Tick(5)
	if x.HasEntity(rock) {
		t.Fatal("rock still simulated after its chunk was deactivated")
	}
	if files := testutil.ChunkFiles(t); len(files) != 0 {
		t.Fatalf("chunk files %v written before the next commit", files)
	}

	s.MovePlayer(testutil.PlayerStart)
	s.Tick(5)
	if !x.HasEntity(rock) {
		t.Fatal("rock didn't come back from memory")
	}
}

func TestEntityQueuedForUnloadedChunkRestoredBeforeCommit(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	x := s.ECS
	s.Commit()
	e := testutil.AddMovable(t, x, nearPlayer, true)
	testutil.SetPos(t, x, e, farAway)
	s.Tick(3)
	if x.HasEntity(e) {
		t.Fatal("entity in an unloaded chunk still simulated")
	}

	s.MovePlayer(farAway)
	s.Tick(5)
	if !x.HasEntity(e) {
		t.Fatal("queued entity didn't come back with its chunk")
	}
}

func TestSpawnRefusedOutsideActiveChunks(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)

	if _, err := s.ECS.AddNewEntity(ecs.NewTransformComponent(elsewhere, 1, 0)); err == nil {
		t.Fatal("spawn outside the active chunks was allowed")
	}
	if _, err := s.ECS.AddNewEntity(ecs.NewTransformComponent(nearPlayer, 1, 0)); err != nil {
		t.Fatalf("spawn inside the active chunks refused: %v", err)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
