package testutil

import (
	"ebittest/content"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tilesystem"
	"ebittest/utils"
	"ebittest/world"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	AtlasPath          = "assets/tiles/atlas.json"
	PlayerLoaderRadius = 3
)

var PlayerStart = utils.Vec2f{X: 100, Y: 100}

func NewECS(t testing.TB) *ecs.ECSContainer {
	t.Helper()

	x := ecs.NewECSContainer()
	if err := content.Register(x); err != nil {
		t.Fatal(err)
	}
	x.TickState = *common.NewTickState()

	return x
}

func Add(t testing.TB, x *ecs.ECSContainer, comps ...ecs.Component) common.EntityId {
	t.Helper()

	e, err := x.AddNewEntity(comps...)
	if err != nil {
		t.Fatal(err)
	}

	return e
}

func AddMovable(t testing.TB, x *ecs.ECSContainer, pos utils.Vec2f, persistent bool, extra ...ecs.Component) common.EntityId {
	t.Helper()

	comps := []ecs.Component{
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(pos, 1, 0),
		ecs.NewDefaultVelocityComponent(),
	}
	if persistent {
		comps = append(comps, ecs.NewPersistentComponent())
	}

	return Add(t, x, append(comps, extra...)...)
}

func Attach(t testing.TB, x *ecs.ECSContainer, child, parent common.EntityId, linkKind ecs.LinkKindEnum) {
	t.Helper()

	if err := x.HierarchyManager.Attach(child, parent, linkKind, x); err != nil {
		t.Fatal(err)
	}
}

func SetPos(t testing.TB, x *ecs.ECSContainer, e common.EntityId, pos utils.Vec2f) {
	t.Helper()

	if err := x.TransformManager.SetLocalPos(e, pos, x); err != nil {
		t.Fatal(err)
	}
}

func WorldPos(t testing.TB, x *ecs.ECSContainer, e common.EntityId) utils.Vec2f {
	t.Helper()

	pos, err := x.TransformManager.GetWorldPos(e, x)
	if err != nil {
		t.Fatal(err)
	}

	return pos
}

func RemoveScheduled(t testing.TB, x *ecs.ECSContainer) {
	t.Helper()

	if err := x.RemoveScheduledEntities(); err != nil {
		t.Fatal(err)
	}
}

// Copies the entities into a fresh container the way loading does, through their DTOs
func Restore(t testing.TB, members map[common.EntityId][]ecs.ComponentDto) *ecs.ECSContainer {
	t.Helper()

	y := NewECS(t)
	for e, dtos := range members {
		comps := make([]ecs.Component, 0, len(dtos))
		for _, dto := range dtos {
			comp, err := ecs.DtoToComponent(dto)
			if err != nil {
				t.Fatal(err)
			}
			comps = append(comps, comp)
		}

		if err := y.AddEntityWithId(e, comps...); err != nil {
			t.Fatal(err)
		}
	}

	return y
}

func ApproxEqual(a, b utils.Vec2f) bool {
	const epsilon = 1e-9
	return a.Subtract(b).Length() < epsilon
}

// Worlds are saved and assets loaded relative to the working directory, so each test gets a directory of its own
// with a copy of the assets
func TempWorld(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.CopyFS(filepath.Join(dir, "assets"), os.DirFS(filepath.Join(repoRoot(), "assets"))); err != nil {
		t.Fatal(err)
	}

	t.Chdir(dir)
	return dir
}

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func ChunkFiles(t testing.TB) []string {
	t.Helper()

	entries, err := os.ReadDir(data.ChunkDataDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	var files []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "chunk_") {
			files = append(files, entry.Name())
		}
	}

	return files
}

type Session struct {
	t       testing.TB
	ECS     *ecs.ECSContainer
	Chunks  *tilesystem.ChunkContainer
	Player  common.EntityId
	Resumed bool
}

// Starts or resumes the world in the working directory the way the game does, up to the first simulated tick.
// A new world gets a minimal player instead of the game's, so tests don't depend on how that one is set up.
func StartSession(t testing.TB) *Session {
	t.Helper()

	x := NewECS(t)
	cc := tilesystem.NewChunkContainer()
	if err := cc.GenerateTileAtlasFromJson(AtlasPath); err != nil {
		t.Fatal(err)
	}
	x.SetSimulatedPosCheck(cc.IsActiveWorldPos)

	h, exists, err := world.Load(x)
	if err != nil {
		t.Fatal(err)
	}

	s := &Session{t: t, ECS: x, Chunks: cc, Resumed: exists}
	if exists {
		if err := tilesystem.RecoverChunkFiles(h.Generation); err != nil {
			t.Fatal(err)
		}
		cc.SetGeneration(h.Generation)
		s.Player = h.PlayerEntity
	} else {
		if err := world.Reset(); err != nil {
			t.Fatal(err)
		}
		s.Player = Add(t, x,
			ecs.NewHierarchyComponent(),
			ecs.NewTransformComponent(PlayerStart, 1, 0),
			ecs.NewDefaultVelocityComponent(),
			ecs.NewChunkLoaderComponent(PlayerLoaderRadius),
			ecs.NewPersistentComponent(),
		)
	}

	s.finishLoading()
	return s
}

func (s *Session) finishLoading() {
	s.t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		ready, err := s.Chunks.Prepare(s.ECS)
		if err != nil {
			s.t.Fatal(err)
		}
		if ready {
			s.Tick(1)
			return
		}
		time.Sleep(time.Millisecond)
	}

	s.t.Fatal("loading never finished")
}

func (s *Session) Tick(n int) {
	s.t.Helper()

	for range n {
		if err := s.Chunks.Tick(s.ECS); err != nil {
			s.t.Fatal(err)
		}
		RemoveScheduled(s.t, s.ECS)
		s.ECS.TickIdx++
	}
}

func (s *Session) MovePlayer(pos utils.Vec2f) {
	s.t.Helper()
	SetPos(s.t, s.ECS, s.Player, pos)
}

func (s *Session) Commit() {
	s.t.Helper()

	h, err := world.Snapshot(s.Player, s.ECS)
	if err != nil {
		s.t.Fatal(err)
	}
	if err := s.Chunks.CommitAndWait(h, s.ECS); err != nil {
		s.t.Fatal(err)
	}
}
