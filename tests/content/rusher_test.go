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

// Moving the slime by its input each time it rushes, until it stands still or ticks run out. Returns its state then.
func rushUntilStill(t *testing.T, x *ecs.ECSContainer, slime common.EntityId, ticks int) inputsources.RusherState {
	t.Helper()
	for range ticks {
		input, state := rush(t, x, slime)
		if reflect.DeepEqual(input, ecs.InputState{}) {
			return state
		}
		if err := x.SetWorldPos(slime, testutil.WorldPos(t, x, slime).Add(utils.Vec2f{X: input.Analog1X, Y: input.Analog1Y})); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("the slime didn't stop")
	return 0
}

func TestTheRusherSearchesAsCloseAsItCanGetThenIdles(t *testing.T) {
	water := pond(-2, 2, -2, 2)
	delete(water, utils.Vec2i{})
	x, player, slime := rusherBy(t, water, utils.Vec2i{X: 6})
	rush(t, x, slime)
	if err := x.SetWorldPos(player, utils.TilePosToWorldPos(utils.Vec2i{X: -38})); err != nil {
		t.Fatal(err)
	}

	if state := rushUntilStill(t, x, slime, 200); state != inputsources.Rusher_Idle {
		t.Fatalf("with the player last seen on an island, the slime stopped in state %v, want idle", state)
	}
	if at := utils.WorldPosToTilePos(testutil.WorldPos(t, x, slime)); at != (utils.Vec2i{X: 3}) {
		t.Fatalf("the slime stopped on %v, want the shore closest to the island", at)
	}
}

// Counting how many tiles walking speeds are asked for, which a path search asks for every tile it looks at
type countedPondWorld struct {
	pondWorld
	asked *int
}

func (w countedPondWorld) WalkSpeedPercent(t utils.Vec2i) int {
	*w.asked++
	return w.pondWorld.WalkSpeedPercent(t)
}

func TestTheRusherWaitsAsCloseAsItCanGetAndOnlyLooksForAWayAgainNowAndThen(t *testing.T) {
	water := pond(-2, 2, -2, 2)
	delete(water, utils.Vec2i{})
	x, _, slime := rusherBy(t, water, utils.Vec2i{X: 6})
	asked := 0
	x.SetTerrain(countedPondWorld{pondWorld: pondWorld{Terrain: x.Terrain().(pondWorld).Terrain, water: water}, asked: &asked})

	if state := rushUntilStill(t, x, slime, 200); state != inputsources.Rusher_Follow {
		t.Fatalf("with the player on an island, the slime stopped in state %v, want still following", state)
	}
	if at := utils.WorldPosToTilePos(testutil.WorldPos(t, x, slime)); at != (utils.Vec2i{X: 3}) {
		t.Fatalf("the slime stopped on %v, want the shore closest to the island", at)
	}

	in, _ := x.Inputs.Get(slime)
	searched := 0
	for tick := x.TickIdx; tick < x.TickIdx+4*data.PathRetryTicks; tick++ {
		asked = 0
		if _, err := inputsources.RusherInputSource(slime, tick, in.Params(), x); err != nil {
			t.Fatal(err)
		}
		if asked > 1000 {
			searched++
		}
	}
	if searched < 3 || searched > 5 {
		t.Fatalf("over %d ticks the slime looked for a way %d times, want about every %d ticks", 4*data.PathRetryTicks, searched, data.PathRetryTicks)
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

var doorTile = utils.Vec2i{X: 3}

// The pond world split by a wall down x = 3, with a door through it on doorTile
type doorWorld struct {
	pondWorld
	open    bool
	faction data.FactionEnum
}

func (w *doorWorld) wall(t utils.Vec2i) bool { return t.X == doorTile.X && t != doorTile }
func (w *doorWorld) IsOpaqueTile(t utils.Vec2i) bool {
	return w.pondWorld.IsOpaqueTile(t) || w.wall(t) || (t == doorTile && !w.open)
}
func (w *doorWorld) WalkSpeedPercent(t utils.Vec2i) int {
	if w.wall(t) {
		return 0
	}
	return w.pondWorld.WalkSpeedPercent(t)
}
func (w *doorWorld) ClosedDoor(t utils.Vec2i) (bool, data.FactionEnum, bool) {
	if t != doorTile || w.open {
		return false, data.Faction_None, false
	}
	return true, w.faction, true
}

func rusherByADoor(t *testing.T, slimePos utils.Vec2f, faction data.FactionEnum) (*ecs.ECSContainer, common.EntityId, common.EntityId, *doorWorld) {
	t.Helper()
	x, player, slime := playerAndSlime(t, slimePos)
	world := &doorWorld{pondWorld: pondWorld{Terrain: x.Terrain()}, faction: faction}
	x.SetTerrain(world)
	return x, player, slime, world
}

func TestTheRusherOnlyGoesThroughClosedDoorsOfItsOwn(t *testing.T) {
	for _, c := range []struct {
		name    string
		faction data.FactionEnum
		through bool
	}{
		{"the colony's", data.Faction_Colony, false},
		{"no one's", data.Faction_None, false},
		{"its own", data.Faction_Monsters, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			x, _, slime, world := rusherByADoor(t, utils.TilePosToWorldPos(utils.Vec2i{X: 6}), c.faction)
			world.open = true
			if _, state := rush(t, x, slime); state != inputsources.Rusher_Follow {
				t.Fatalf("through the open door, the slime is in state %v, want following the player", state)
			}

			world.open = false
			rushUntilStill(t, x, slime, 200)
			if through := testutil.WorldPos(t, x, slime).X < utils.TilePosToWorldPos(doorTile).X; through != c.through {
				t.Fatalf("with the door closed, the slime went through %v, want %v", through, c.through)
			}
		})
	}
}

func TestSomethingOnAClosedDoorsTileSeesAndIsSeenFromItsSideOfIt(t *testing.T) {
	door := utils.TilePosToWorldPos(doorTile)
	for _, c := range []struct {
		name         string
		slime, other utils.Vec2f
		seen         bool
	}{
		{"seeing from behind it", door.Add(utils.Vec2f{X: 2}), utils.Vec2f{}, false},
		{"seeing from in front of it", door.Add(utils.Vec2f{X: -2}), utils.Vec2f{}, true},
		{"seen behind it", utils.TilePosToWorldPos(utils.Vec2i{X: 6}), door.Add(utils.Vec2f{X: -2}), false},
		{"seen in front of it", utils.TilePosToWorldPos(utils.Vec2i{X: 6}), door.Add(utils.Vec2f{X: 2}), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			x, player, slime, _ := rusherByADoor(t, c.slime, data.Faction_Colony)
			if err := x.SetWorldPos(player, c.other); err != nil {
				t.Fatal(err)
			}
			if _, state := rush(t, x, slime); (state == inputsources.Rusher_Follow) != c.seen {
				t.Fatalf("the slime is in state %v, want it seeing the player %v", state, c.seen)
			}
		})
	}
}
