package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/systems/visionsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestTileFlagsAreWhatEachTileSaysOnItsOwn(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)

	// Across the corner of four chunks, and across the edge of the active ones
	for _, from := range []utils.Vec2i{{X: data.ChunkSize - 10, Y: data.ChunkSize - 10}, {X: 2*data.ChunkSize - 10, Y: -5}} {
		const w, h = 20, 15
		opaque, sky := make([]bool, w*h), make([]bool, w*h)
		s.Chunks.TileFlags(from, w, h, opaque, sky)
		for i := range w * h {
			tile := utils.Vec2i{X: from.X + i%w, Y: from.Y + i/w}
			if opaque[i] != s.Chunks.IsOpaqueTile(tile) || sky[i] != s.Chunks.IsOpenToSky(tile) {
				t.Fatalf("%v is opaque %v and open to the sky %v, but on its own says %v and %v",
					tile, opaque[i], sky[i], s.Chunks.IsOpaqueTile(tile), s.Chunks.IsOpenToSky(tile))
			}
		}
	}
}

// A light map worked out for the first time, to compare with one that was kept
func freshLight(s *testutil.Session, center utils.Vec2i, radius int) *ecs.LightMap {
	m := &ecs.LightMap{}
	s.ECS.LightAround(center, radius, m)
	return m
}

func sameLight(t *testing.T, kept, fresh *ecs.LightMap, center utils.Vec2i, radius int) {
	t.Helper()
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
			if kept.At(tile) != fresh.At(tile) {
				t.Fatalf("the kept light at %v is %d, but worked out again it's %d", tile, kept.At(tile), fresh.At(tile))
			}
		}
	}
}

func TestLightIsOnlyWorkedOutAgainWhenSomethingItDependsOnChanges(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	center := utils.WorldPosToTilePos(testutil.PlayerStart)
	const radius = 20
	m := &ecs.LightMap{}

	if !s.ECS.LightAround(center, radius, m) {
		t.Fatal("light worked out for the first time didn't count as changed")
	}
	if s.ECS.LightAround(center, radius, m) {
		t.Fatal("light was worked out again with nothing changed")
	}

	rock := findTile(t, s, func(tile utils.Vec2i) bool {
		return isRock(tile) && utils.Dist2(tile, center) <= radius*radius
	})
	if ok, _ := s.Chunks.DamageTile(rock, 1000); !ok {
		t.Fatal("the rock couldn't be mined")
	}
	if !s.ECS.LightAround(center, radius, m) {
		t.Fatal("light wasn't worked out again after rock nearby was mined")
	}
	sameLight(t, m, freshLight(s, center, radius), center, radius)

	s.ECS.TickIdx = data.DayTicks + data.DuskTicks/2
	if !s.ECS.LightAround(center, radius, m) {
		t.Fatal("light wasn't worked out again as dusk fell")
	}
	sameLight(t, m, freshLight(s, center, radius), center, radius)

	emitter, err := ecs.NewLightEmitterComponent(150, 10)
	if err != nil {
		t.Fatal(err)
	}
	holder := testutil.AddMovable(t, s.ECS, utils.TilePosToWorldPos(center), false)
	torch := testutil.AddMovable(t, s.ECS, utils.Vec2f{}, false, emitter)
	testutil.Attach(t, s.ECS, torch, holder, ecs.Link_Held)
	if !s.ECS.LightAround(center, radius, m) {
		t.Fatal("light wasn't worked out again for a torch carried into it")
	}
	if s.ECS.LightAround(center, radius, m) {
		t.Fatal("light was worked out again with the torch standing still")
	}
	testutil.SetPos(t, s.ECS, holder, utils.TilePosToWorldPos(center).Add(utils.Vec2f{X: 3 * data.TileSize}))
	if !s.ECS.LightAround(center, radius, m) {
		t.Fatal("light wasn't worked out again for a torch carried further")
	}
	sameLight(t, m, freshLight(s, center, radius), center, radius)

	if !s.ECS.LightAround(utils.Vec2i{X: center.X + 1, Y: center.Y}, radius, m) {
		t.Fatal("light wasn't worked out again around another tile")
	}
}

func TestVisionIsOnlyUpdatedWhenDueAndWhenSomethingChanged(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	viewer := testutil.AddMovable(t, s.ECS, testutil.PlayerStart, false, ecs.NewVisionComponent(10, 3))
	v, _ := s.ECS.Visions.Get(viewer)

	if !v.IsDue(viewer, 1) {
		t.Fatal("a vision that has never been updated isn't due")
	}
	visionsystem.Tick(s.ECS)
	due := 0
	for tick := range uint64(9) {
		if v.IsDue(viewer, tick) {
			due++
		}
	}
	if due != 3 {
		t.Fatalf("a vision updated every 3 ticks was due %d times in 9 ticks", due)
	}

	origin, _ := v.VisibleArea()
	s.ECS.TickIdx = 0
	for !v.IsDue(viewer, s.ECS.TickIdx) {
		s.ECS.TickIdx++
	}
	updates := v.Updates()
	visionsystem.Tick(s.ECS)
	if v.Updates() != updates {
		t.Fatal("on its turn, a vision with nothing changed was updated again")
	}
	testutil.SetPos(t, s.ECS, viewer, utils.TilePosToWorldPos(utils.Vec2i{X: origin.X + 2, Y: origin.Y}))
	visionsystem.Tick(s.ECS)
	if moved, _ := v.VisibleArea(); moved != (utils.Vec2i{X: origin.X + 2, Y: origin.Y}) || v.Updates() != updates+1 {
		t.Fatalf("on its turn, a vision that moved is still seeing from %v", moved)
	}
}
