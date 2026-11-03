package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

// Light everywhere at the given level, without walls
func evenLight(light uint8) func(utils.Vec2i) uint8 {
	return func(utils.Vec2i) uint8 { return light }
}

func noWalls(utils.Vec2i) bool { return false }

func TestTilesAreSeenAsFarAsTheirLightReaches(t *testing.T) {
	for _, c := range []struct {
		name      string
		light     uint8
		seen, not utils.Vec2i
	}{
		{"full light", data.MaxLight, utils.Vec2i{X: data.VisionMaxRangeTiles}, utils.Vec2i{X: data.VisionMaxRangeTiles, Y: 1}},
		{"night sky light", data.NightSkyLight, utils.Vec2i{X: 12}, utils.Vec2i{X: 13}},
		{"night sky light, diagonally", data.NightSkyLight, utils.Vec2i{X: 8, Y: 8}, utils.Vec2i{X: 9, Y: 9}},
		{"a quarter of full light", data.MaxLight / 4, utils.Vec2i{Y: -15}, utils.Vec2i{Y: -16}},
	} {
		v := ecs.NewVisionComponent()
		v.Update(utils.Vec2i{}, noWalls, evenLight(c.light))
		if !v.CanSee(c.seen) || v.CanSee(c.not) {
			t.Errorf("%s: sees %v %v and %v %v, want only the first", c.name, c.seen, v.CanSee(c.seen), c.not, v.CanSee(c.not))
		}
	}
}

func TestNothingIsSeenInTheDarkNotEvenTheViewersOwnTile(t *testing.T) {
	v := ecs.NewVisionComponent()
	v.Update(utils.Vec2i{}, noWalls, evenLight(0))
	for _, tile := range []utils.Vec2i{{}, {X: 1}} {
		if v.CanSee(tile) {
			t.Fatalf("sees %v in the dark", tile)
		}
	}
}

func TestALitTileIsSeenFromAfarOnlyInLineOfSight(t *testing.T) {
	lamp, hidden := utils.Vec2i{X: 40}, utils.Vec2i{X: 40, Y: 20}
	wall := func(t utils.Vec2i) bool { return t.X == 30 && t.Y >= 5 }
	lit := func(t utils.Vec2i) uint8 {
		if t == lamp || t == hidden {
			return data.MaxLight
		}
		return 0
	}

	v := ecs.NewVisionComponent()
	v.Update(utils.Vec2i{}, wall, lit)
	if !v.CanSee(lamp) {
		t.Fatal("a brightly lit tile 40 tiles away in line of sight isn't seen")
	}
	if v.CanSee(hidden) {
		t.Fatal("a brightly lit tile behind a wall is seen")
	}
	if v.CanSee(utils.Vec2i{X: 20}) {
		t.Fatal("a dark tile between the viewer and the lit one is seen")
	}
}

func TestVisionFollowsTheViewerAtTheNextUpdate(t *testing.T) {
	v := ecs.NewVisionComponent()
	v.Update(utils.Vec2i{}, noWalls, evenLight(data.NightSkyLight))
	if v.CanSee(utils.Vec2i{X: 100}) {
		t.Fatal("sees a tile far away before moving there")
	}
	v.Update(utils.Vec2i{X: 100}, noWalls, evenLight(data.NightSkyLight))
	if !v.CanSee(utils.Vec2i{X: 100}) || v.CanSee(utils.Vec2i{}) {
		t.Fatal("what's visible didn't follow the viewer")
	}
}

func TestVisionIsNotSaved(t *testing.T) {
	x := testutil.NewECS(t)
	e := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewVisionComponent())
	v, _ := x.Visions.Get(e)
	v.Update(utils.Vec2i{}, noWalls, evenLight(data.MaxLight))

	members, err := x.GetCompositionComponents(e)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)

	restored, ok := y.Visions.Get(e)
	if !ok {
		t.Fatal("the vision wasn't restored")
	}
	if restored.CanSee(utils.Vec2i{}) {
		t.Fatal("restored vision sees something before its first update")
	}
}
