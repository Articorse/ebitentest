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
		{"full light", data.MaxLight, utils.Vec2i{X: data.PlayerVisionRangeTiles}, utils.Vec2i{X: data.PlayerVisionRangeTiles, Y: 1}},
		{"night sky light", data.NightSkyLight, utils.Vec2i{X: 12}, utils.Vec2i{X: 13}},
		{"night sky light, diagonally", data.NightSkyLight, utils.Vec2i{X: 8, Y: 8}, utils.Vec2i{X: 9, Y: 9}},
		{"a quarter of full light", data.MaxLight / 4, utils.Vec2i{Y: -15}, utils.Vec2i{Y: -16}},
	} {
		v := ecs.NewVisionComponent(data.PlayerVisionRangeTiles, 1)
		v.Update(utils.Vec2i{}, data.PlayerVisionRangeTiles, noWalls, evenLight(c.light))
		if !v.CanSee(c.seen) || v.CanSee(c.not) {
			t.Errorf("%s: sees %v %v and %v %v, want only the first", c.name, c.seen, v.CanSee(c.seen), c.not, v.CanSee(c.not))
		}
	}
}

func TestNothingIsSeenInTheDarkNotEvenTheViewersOwnTile(t *testing.T) {
	v := ecs.NewVisionComponent(data.PlayerVisionRangeTiles, 1)
	v.Update(utils.Vec2i{}, data.PlayerVisionRangeTiles, noWalls, evenLight(0))
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

	v := ecs.NewVisionComponent(data.PlayerVisionRangeTiles, 1)
	v.Update(utils.Vec2i{}, data.PlayerVisionRangeTiles, wall, lit)
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
	v := ecs.NewVisionComponent(data.PlayerVisionRangeTiles, 1)
	v.Update(utils.Vec2i{}, data.PlayerVisionRangeTiles, noWalls, evenLight(data.NightSkyLight))
	if v.CanSee(utils.Vec2i{X: 100}) {
		t.Fatal("sees a tile far away before moving there")
	}
	v.Update(utils.Vec2i{X: 100}, data.PlayerVisionRangeTiles, noWalls, evenLight(data.NightSkyLight))
	if !v.CanSee(utils.Vec2i{X: 100}) || v.CanSee(utils.Vec2i{}) {
		t.Fatal("what's visible didn't follow the viewer")
	}
}

func TestOnlyTheVisionsBaseRangeIsSaved(t *testing.T) {
	x := testutil.NewECS(t)
	e := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewVisionComponent(23, 1))
	v, _ := x.Visions.Get(e)
	v.Update(utils.Vec2i{}, data.PlayerVisionRangeTiles, noWalls, evenLight(data.MaxLight))

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
	if restored.BaseRangeTiles() != 23 {
		t.Fatalf("the restored vision's base range is %d, want 23", restored.BaseRangeTiles())
	}
}

func TestVisionSeesOnlyWithinItsRange(t *testing.T) {
	v := ecs.NewVisionComponent(data.PlayerVisionRangeTiles, 1)
	v.Update(utils.Vec2i{X: 5}, 10, noWalls, evenLight(data.MaxLight))
	if origin, radius := v.VisibleArea(); origin != (utils.Vec2i{X: 5}) || radius != 10 {
		t.Fatalf("the visible area is %d around %v, want 10 around (5, 0)", radius, origin)
	}
	if !v.CanSee(utils.Vec2i{X: 15}) || !v.CanSee(utils.Vec2i{X: -5}) || v.CanSee(utils.Vec2i{X: 16}) || v.CanSee(utils.Vec2i{X: 5, Y: 11}) {
		t.Fatal("with a range of 10 in full light, it doesn't see exactly 10 tiles away")
	}

	v.Update(utils.Vec2i{}, 32, noWalls, evenLight(data.MaxLight/2))
	if !v.CanSee(utils.Vec2i{X: 15}) || v.CanSee(utils.Vec2i{X: 16}) {
		t.Fatal("with a range of 32 in half light, it doesn't see 15 tiles away and no further")
	}

	v.Update(utils.Vec2i{}, 0, noWalls, evenLight(data.MaxLight))
	if v.CanSee(utils.Vec2i{}) {
		t.Fatal("with a range of 0, it sees its own tile")
	}
}
