package testutil

import (
	"ebittest/content"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tilesystem"
	"ebittest/utils"
	"ebittest/world"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	AtlasPath        = "assets/tiles/atlas.json"
	FeatureAtlasPath = "assets/features/features.json"
	ItemAtlasPath    = "assets/items/items.json"
	BodyAtlasPath    = "assets/bodies/bodies.json"
	// Small, since every chunk it loads is generated, and chunks are big
	PlayerLoaderRadius = 1
	// Fixed rather than random like the game's, so that a failing test fails the same way every time
	WorldSeed uint64 = 0x5EED
)

var PlayerStart = utils.Vec2f{X: 100, Y: 100}

func NewECS(t testing.TB) *ecs.ECSContainer {
	t.Helper()

	x := ecs.NewECSContainer()
	if err := content.Register(x); err != nil {
		t.Fatal(err)
	}
	if err := x.SetItemAtlas(ItemAtlas(t)); err != nil {
		t.Fatal(err)
	}
	bodies, err := ecs.LoadBodyAtlasFromJson(filepath.Join(RepoRoot(), BodyAtlasPath))
	if err != nil {
		t.Fatal(err)
	}
	x.SetBodyAtlas(bodies)
	x.TickState = *common.NewTickState()

	return x
}

// From the repository, since tests run in their own package's directory unless they've moved to a temporary world
func ItemAtlas(t testing.TB) *ecs.ItemAtlas {
	t.Helper()

	atlas, err := ecs.LoadItemAtlasFromJson(filepath.Join(RepoRoot(), ItemAtlasPath))
	if err != nil {
		t.Fatal(err)
	}
	return atlas
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

	if err := x.Attach(child, parent, linkKind); err != nil {
		t.Fatal(err)
	}
}

func SetPos(t testing.TB, x *ecs.ECSContainer, e common.EntityId, pos utils.Vec2f) {
	t.Helper()

	tr, ok := x.Transforms.Get(e)
	if !ok {
		t.Fatalf("entity %d has no transform", e)
	}
	tr.SetLocalPos(pos)
}

func WorldPos(t testing.TB, x *ecs.ECSContainer, e common.EntityId) utils.Vec2f {
	t.Helper()

	pos, err := x.WorldPos(e)
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
	y.SetNextEntity(slices.Max(slices.Collect(maps.Keys(members))) + 1)
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
	if err := os.CopyFS(filepath.Join(dir, "assets"), os.DirFS(filepath.Join(RepoRoot(), "assets"))); err != nil {
		t.Fatal(err)
	}

	t.Chdir(dir)
	return dir
}

func RepoRoot() string {
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
	if err := cc.GenerateTileAtlasFromJson(AtlasPath, x.ItemAtlas()); err != nil {
		t.Fatal(err)
	}
	if err := x.ItemAtlas().CheckPlacedTiles(cc.HasTile); err != nil {
		t.Fatal(err)
	}
	if err := cc.GenerateFeatureAtlasFromJson(FeatureAtlasPath, x.ItemAtlas()); err != nil {
		t.Fatal(err)
	}
	if err := x.ItemAtlas().CheckPlacedFeatures(cc.HasFeature); err != nil {
		t.Fatal(err)
	}
	x.SetSimulatedPosCheck(cc.IsActiveWorldPos)
	x.SetTerrain(cc)

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
		cc.SetSeed(h.Seed)
		if err := cc.LoadMemory(); err != nil {
			t.Fatal(err)
		}
		s.Player = h.PlayerEntity
	} else {
		if err := world.Reset(); err != nil {
			t.Fatal(err)
		}
		cc.SetSeed(WorldSeed)
		x.SeedRng(WorldSeed)
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

	if err := s.Chunks.CommitAndWait(s.snapshot(), s.ECS); err != nil {
		s.t.Fatal(err)
	}
}

// Leaves the commit running on the I/O thread, as the game's periodic saves do
func (s *Session) StartCommit() {
	s.t.Helper()

	if err := s.Chunks.StartCommit(s.snapshot(), s.ECS); err != nil {
		s.t.Fatal(err)
	}
}

func (s *Session) snapshot() world.Header {
	s.t.Helper()

	h, err := world.Snapshot(s.Chunks.Seed(), s.Player, s.ECS)
	if err != nil {
		s.t.Fatal(err)
	}
	return h
}

const DummyPart = "core"

// The game's bodies plus a dummy body made of one vital part with the given health, so that damage to it is exact.
// Every dummy body in the container has the health given last, since they're all of the same kind.
func DummyBody(t testing.TB, x *ecs.ECSContainer, health int) *ecs.Body {
	t.Helper()

	f, err := os.ReadFile(filepath.Join(RepoRoot(), BodyAtlasPath))
	if err != nil {
		t.Fatal(err)
	}
	var atlas map[string]any
	if err := json.Unmarshal(f, &atlas); err != nil {
		t.Fatal(err)
	}
	const dummyKind = 999
	atlas["bodies"] = append(atlas["bodies"].([]any), map[string]any{
		"id":    dummyKind,
		"name":  "Dummy",
		"parts": []any{map[string]any{"name": DummyPart, "maxHealth": health, "coverage": 100, "vital": true}},
	})

	out, err := json.Marshal(atlas)
	if err != nil {
		t.Fatal(err)
	}
	x.SetBodyAtlas(BodyAtlasFrom(t, string(out)))
	return ecs.NewBodyComponent(dummyKind)
}

func BodyAtlasFrom(t testing.TB, atlasJson string) *ecs.BodyAtlas {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bodies.json")
	if err := os.WriteFile(path, []byte(atlasJson), 0o644); err != nil {
		t.Fatal(err)
	}
	atlas, err := ecs.LoadBodyAtlasFromJson(path)
	if err != nil {
		t.Fatal(err)
	}
	return atlas
}

func PartHealth(t testing.TB, x *ecs.ECSContainer, e common.EntityId, part string) int {
	t.Helper()
	health, _, ok := x.PartHealth(e, part)
	if !ok {
		t.Fatalf("entity %d has no part %q", e, part)
	}
	return health
}
