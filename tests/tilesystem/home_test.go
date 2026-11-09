package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestHomeIsMarkedAndUnmarkedTileByTileAcrossChunks(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	first, last := utils.Vec2i{X: data.ChunkSize - 2, Y: -3}, utils.Vec2i{X: data.ChunkSize + 1, Y: 2}

	s.Chunks.SetArea(data.Area_Home, first, last, true)
	for y := first.Y - 1; y <= last.Y+1; y++ {
		for x := first.X - 1; x <= last.X+1; x++ {
			tile := utils.Vec2i{X: x, Y: y}
			want := x >= first.X && x <= last.X && y >= first.Y && y <= last.Y
			if s.Chunks.InArea(data.Area_Home, tile) != want {
				t.Fatalf("%v is home %v, want %v", tile, !want, want)
			}
		}
	}

	s.Chunks.SetArea(data.Area_Home, utils.Vec2i{X: first.X, Y: 0}, last, false)
	if s.Chunks.InArea(data.Area_Home, last) || !s.Chunks.InArea(data.Area_Home, first) {
		t.Fatal("unmarking part of home didn't leave just the rest")
	}

	s.Commit()
	r := testutil.StartSession(t)
	if !r.Chunks.InArea(data.Area_Home, first) || r.Chunks.InArea(data.Area_Home, last) {
		t.Fatal("after restarting, home isn't what it was")
	}
	r.Chunks.SetArea(data.Area_Home, first, last, false)
	if r.Chunks.InArea(data.Area_Home, first) {
		t.Fatal("home couldn't be unmarked entirely")
	}
}

func TestHomeIsOnlyMarkedInsideTheColony(t *testing.T) {
	s, colony, _ := colonistAtWorkWithoutHome(t)
	first, last, _ := s.ECS.ColonyBounds(colony)
	outside := utils.Vec2i{X: first.X - 1, Y: first.Y}

	inputs := map[common.EntityId]ecs.InputState{s.Player: {Area: ecs.AreaMarking{Area: data.Area_Home, First: outside, Last: first}}}
	if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
		t.Fatal(err)
	}
	if !s.Chunks.InArea(data.Area_Home, first) || s.Chunks.InArea(data.Area_Home, outside) {
		t.Fatal("marking home across the colony's edge didn't mark just the colony's side")
	}
	inputs[s.Player] = ecs.InputState{Area: ecs.AreaMarking{Area: data.Area_Home, Erase: true, First: first, Last: last}}
	if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
		t.Fatal(err)
	}
	if s.Chunks.InArea(data.Area_Home, first) {
		t.Fatal("unmarking the whole colony left home")
	}
}

func colonistAtWorkWithoutHome(t *testing.T) (*testutil.Session, common.EntityId, common.EntityId) {
	t.Helper()
	s, colony, colonist := colonistAtWork(t)
	first, last, _ := s.ECS.ColonyBounds(colony)
	s.ECS.MarkArea(data.Area_Home, first, last, false)
	return s, colony, colonist
}

func TestBuildingOnHomeSpreadsItButBuildingElsewhereDoesnt(t *testing.T) {
	s, _, colonist := colonistAtWorkWithoutHome(t)
	onHome := groundAway(t, s, colonist, 6)
	s.ECS.MarkArea(data.Area_Home, onHome, onHome, true)
	s.Chunks.PlaceSite(onHome, data.Blueprint_WoodWall)
	if !live(t, s, 30*data.TPS, isBuilt(s, onHome)) {
		t.Fatal("the colonist didn't build the site on home")
	}

	r := data.HomeSpreadTiles
	for _, tile := range []utils.Vec2i{{X: onHome.X - r, Y: onHome.Y - r}, {X: onHome.X + r, Y: onHome.Y + r}} {
		if !s.Chunks.InArea(data.Area_Home, tile) {
			t.Fatalf("%v, %d tiles from what was built on home, isn't home", tile, r)
		}
	}
	if s.Chunks.InArea(data.Area_Home, utils.Vec2i{X: onHome.X + r + 1, Y: onHome.Y}) {
		t.Fatal("home spread further than it should")
	}

	// The player, as a builder, builds off home; the colonist leaves it alone
	for _, c := range []ecs.Component{
		ecs.NewInventoryComponent(2, 2, 1000),
		ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{}),
		ecs.NewConstructorComponent(3, 1, 250),
	} {
		s.ECS.AddComponent(s.Player, c)
	}
	elsewhere := groundAway(t, s, colonist, 3*r, onHome)
	if s.Chunks.InArea(data.Area_Home, elsewhere) {
		t.Fatal("the ground chosen off home is home")
	}
	builderAt, found := utils.Vec2i{}, false
	for dy := -2; dy <= 2 && !found; dy++ {
		for dx := -2; dx <= 2 && !found; dx++ {
			builderAt = utils.Vec2i{X: elsewhere.X + dx, Y: elsewhere.Y + dy}
			sees := s.ECS.TileInReachFrom([]utils.Vec2i{builderAt}, elsewhere, 3)
			found = builderAt != elsewhere && sees && s.Chunks.WalkSpeedPercent(builderAt) > 0
		}
	}
	if !found {
		t.Fatal("there's nowhere to build from next to the ground off home")
	}
	testutil.SetPos(t, s.ECS, s.Player, utils.TilePosToWorldPos(builderAt))
	s.Chunks.PlaceSite(elsewhere, data.Blueprint_WoodWall)
	s.Chunks.DeliverToSite(elsewhere, data.Item_Wood, wallCost(t))
	for range 10 {
		waitOutTheCooldown(s)
		construct(t, s, elsewhere, false)
	}
	if currentTile(s, elsewhere) != data.Tile_WoodWall || s.Chunks.InArea(data.Area_Home, utils.Vec2i{X: elsewhere.X + 2, Y: elsewhere.Y}) {
		t.Fatal("building off home spread home around it")
	}
}

func TestAreasAreMarkedApartFromEachOtherAndKept(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	home, both, harvest := utils.Vec2i{X: 2, Y: 2}, utils.Vec2i{X: 3, Y: 2}, utils.Vec2i{X: 4, Y: 2}
	s.Chunks.SetArea(data.Area_Home, home, both, true)
	s.Chunks.SetArea(data.Area_Harvest, both, harvest, true)
	s.Chunks.SetArea(data.Area_Home, home, home, false)
	s.Chunks.SetArea(data.Area_Home, home, home, true)

	want := func(chunks interface {
		InArea(data.AreaEnum, utils.Vec2i) bool
	}, when string) {
		for _, c := range []struct {
			tile          utils.Vec2i
			home, harvest bool
		}{{home, true, false}, {both, true, true}, {harvest, false, true}} {
			if chunks.InArea(data.Area_Home, c.tile) != c.home || chunks.InArea(data.Area_Harvest, c.tile) != c.harvest {
				t.Fatalf("%s, %v is home %v and harvest %v, want %v and %v", when, c.tile,
					chunks.InArea(data.Area_Home, c.tile), chunks.InArea(data.Area_Harvest, c.tile), c.home, c.harvest)
			}
		}
	}
	want(s.Chunks, "after marking")
	s.Commit()
	r := testutil.StartSession(t)
	want(r.Chunks, "after restarting")

	r.Chunks.SetArea(data.Area_Harvest, both, harvest, false)
	if r.Chunks.InArea(data.Area_Harvest, both) || !r.Chunks.InArea(data.Area_Home, both) {
		t.Fatal("erasing harvest took the tile out of home, or left it in harvest")
	}
}

func TestHarvestIsOnlyMarkedInsideTheColony(t *testing.T) {
	s, colony, _ := colonistAtWorkWithoutHome(t)
	first, _, _ := s.ECS.ColonyBounds(colony)
	outside := utils.Vec2i{X: first.X - 1, Y: first.Y}
	inputs := map[common.EntityId]ecs.InputState{s.Player: {Area: ecs.AreaMarking{Area: data.Area_Harvest, First: outside, Last: first}}}
	if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
		t.Fatal(err)
	}
	if !s.Chunks.InArea(data.Area_Harvest, first) || s.Chunks.InArea(data.Area_Harvest, outside) || s.Chunks.InArea(data.Area_Home, first) {
		t.Fatal("marking harvest across the colony's edge didn't mark just the colony's side, and just harvest")
	}
}
