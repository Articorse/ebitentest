package content_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"reflect"
	"testing"
)

// Open to the sky within 40 tiles of the origin, with deep water where told, which can be seen across but not walked
type pondWorld struct {
	ecs.Terrain
	water map[utils.Vec2i]bool
}

func (w pondWorld) inside(t utils.Vec2i) bool {
	return utils.AbsInt(t.X) <= 40 && utils.AbsInt(t.Y) <= 40
}
func (w pondWorld) IsOpenToSky(t utils.Vec2i) bool  { return w.inside(t) }
func (w pondWorld) IsOpaqueTile(t utils.Vec2i) bool { return !w.inside(t) }
func (w pondWorld) WalkSpeedPercent(t utils.Vec2i) int {
	if !w.inside(t) || w.water[t] {
		return 0
	}
	return 100
}

func pond(minX, maxX, minY, maxY int) map[utils.Vec2i]bool {
	water := map[utils.Vec2i]bool{}
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			water[utils.Vec2i{X: x, Y: y}] = true
		}
	}
	return water
}

func rusherBy(t *testing.T, water map[utils.Vec2i]bool, slimeTile utils.Vec2i) (*ecs.ECSContainer, common.EntityId, common.EntityId) {
	t.Helper()
	x, player, slime := playerAndSlime(t, utils.TilePosToWorldPos(slimeTile))
	x.SetTerrain(pondWorld{Terrain: x.Terrain(), water: water})
	return x, player, slime
}

func pathOf(x *ecs.ECSContainer, slime common.EntityId) []utils.Vec2i {
	in, _ := x.Inputs.Get(slime)
	return in.Params().(*inputsources.InputRusherParams).CurrentPath()
}

func TestTheRusherGoesStraightWhileNothingsInTheWay(t *testing.T) {
	x, _, slime := rusherBy(t, nil, utils.Vec2i{X: 5, Y: 2})
	input, state := rush(t, x, slime)
	want := utils.Vec2f{X: -5, Y: -2}.Normalized()
	if state != inputsources.Rusher_Follow || input.Analog1X != want.X || input.Analog1Y != want.Y {
		t.Fatalf("in the open, the slime is in state %v heading %v, %v; want straight at the player", state, input.Analog1X, input.Analog1Y)
	}
	if path := pathOf(x, slime); len(path) != 0 {
		t.Fatalf("in the open, the slime follows a path %v", path)
	}
}

func TestTheRusherWalksAroundWhatItCanSeeAcrossButNotWalkThrough(t *testing.T) {
	water := pond(2, 3, -4, 4)
	x, player, slime := rusherBy(t, water, utils.Vec2i{X: 6})

	for tick := 0; ; tick++ {
		if tick == 400 {
			t.Fatalf("the slime didn't get round the pond to the player, and is at %v", testutil.WorldPos(t, x, slime))
		}
		input, state := rush(t, x, slime)
		if state != inputsources.Rusher_Follow {
			t.Fatalf("the slime stopped following, in state %v", state)
		}
		if path := pathOf(x, slime); len(path) > 0 && path[0] != utils.WorldPosToTilePos(testutil.WorldPos(t, x, slime)) {
			t.Fatalf("the slime's path %v doesn't start on its tile", path)
		}
		for _, step := range pathOf(x, slime) {
			if water[step] {
				t.Fatalf("the slime's path %v goes through the pond", pathOf(x, slime))
			}
		}

		pos := testutil.WorldPos(t, x, slime).Add(utils.Vec2f{X: input.Analog1X, Y: input.Analog1Y})
		tile := utils.WorldPosToTilePos(pos)
		if water[tile] {
			t.Fatalf("the slime walked into the pond at %v", tile)
		}
		if err := x.SetWorldPos(slime, pos); err != nil {
			t.Fatal(err)
		}
		if tile == (utils.Vec2i{}) {
			break
		}
	}

	// The player moving to another tile is found a new path to
	if err := x.SetWorldPos(slime, utils.TilePosToWorldPos(utils.Vec2i{X: 6})); err != nil {
		t.Fatal(err)
	}
	rush(t, x, slime)
	if err := x.SetWorldPos(player, utils.TilePosToWorldPos(utils.Vec2i{Y: 1})); err != nil {
		t.Fatal(err)
	}
	rush(t, x, slime)
	if path := pathOf(x, slime); len(path) == 0 || path[len(path)-1] != (utils.Vec2i{Y: 1}) {
		t.Fatalf("after the player moved to (0, 1), the slime's path is %v", path)
	}

	if err := x.SetWorldPos(player, utils.TilePosToWorldPos(utils.Vec2i{X: -38})); err != nil {
		t.Fatal(err)
	}
	if _, state := rush(t, x, slime); state != inputsources.Rusher_Search {
		t.Fatalf("with the player out of sight, the slime is in state %v, want it searching", state)
	}
	if path := pathOf(x, slime); len(path) == 0 || path[len(path)-1] != (utils.Vec2i{Y: 1}) {
		t.Fatalf("searching, the slime's path is %v, want it round the pond to where it last saw the player", path)
	}
}

func TestTheRusherGivesUpSearchingWhereItCantGo(t *testing.T) {
	water := pond(-2, 2, -2, 2)
	delete(water, utils.Vec2i{})
	x, player, slime := rusherBy(t, water, utils.Vec2i{X: 6})
	rush(t, x, slime)
	if err := x.SetWorldPos(player, utils.TilePosToWorldPos(utils.Vec2i{X: -38})); err != nil {
		t.Fatal(err)
	}

	if input, state := rush(t, x, slime); state != inputsources.Rusher_Idle || !reflect.DeepEqual(input, ecs.InputState{}) {
		t.Fatalf("with the player last seen on an island, the slime is in state %v heading %v, %v; want it idle", state, input.Analog1X, input.Analog1Y)
	}
}

func TestTheRusherStandsStillWhenNoPathReachesItsTarget(t *testing.T) {
	water := pond(-2, 2, -2, 2)
	delete(water, utils.Vec2i{})
	x, _, slime := rusherBy(t, water, utils.Vec2i{X: 6})

	input, state := rush(t, x, slime)
	if state != inputsources.Rusher_Follow || !reflect.DeepEqual(input, ecs.InputState{}) {
		t.Fatalf("with the player on an island, the slime is in state %v heading %v, %v; want it following but still", state, input.Analog1X, input.Analog1Y)
	}
}

func TestTheRusherGoesForTheClosestHostileItSees(t *testing.T) {
	x, player, slime := rusherBy(t, nil, utils.Vec2i{X: 6})
	colonist := testutil.AddMovable(t, x, utils.TilePosToWorldPos(utils.Vec2i{X: 9}), false,
		ecs.NewFactionComponent(data.Faction_Colony), ecs.NewBodyComponent(data.Body_Humanoid))
	target := func() common.EntityId {
		in, _ := x.Inputs.Get(slime)
		return in.Params().(*inputsources.InputRusherParams).Target
	}

	if input, _ := rush(t, x, slime); target() != colonist || input.Analog1X <= 0 {
		t.Fatalf("with a colonist 3 tiles away and the player 6, the slime goes for %d heading %v, want the colonist %d", target(), input.Analog1X, colonist)
	}
	if err := x.SetWorldPos(colonist, utils.TilePosToWorldPos(utils.Vec2i{X: 14})); err != nil {
		t.Fatal(err)
	}
	if input, _ := rush(t, x, slime); target() != player || input.Analog1X >= 0 {
		t.Fatalf("once the player is closer, the slime goes for %d heading %v, want the player %d", target(), input.Analog1X, player)
	}

	x.SetUntargetable(player, true)
	if _, state := rush(t, x, slime); target() != colonist || state != inputsources.Rusher_Follow {
		t.Fatalf("with the player untargetable, the slime goes for %d in state %v, want the colonist %d", target(), state, colonist)
	}
}
