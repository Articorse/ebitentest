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

	s.Chunks.SetHome(first, last, true)
	for y := first.Y - 1; y <= last.Y+1; y++ {
		for x := first.X - 1; x <= last.X+1; x++ {
			tile := utils.Vec2i{X: x, Y: y}
			want := x >= first.X && x <= last.X && y >= first.Y && y <= last.Y
			if s.Chunks.IsHome(tile) != want {
				t.Fatalf("%v is home %v, want %v", tile, !want, want)
			}
		}
	}

	s.Chunks.SetHome(utils.Vec2i{X: first.X, Y: 0}, last, false)
	if s.Chunks.IsHome(last) || !s.Chunks.IsHome(first) {
		t.Fatal("unmarking part of home didn't leave just the rest")
	}

	s.Commit()
	r := testutil.StartSession(t)
	if !r.Chunks.IsHome(first) || r.Chunks.IsHome(last) {
		t.Fatal("after restarting, home isn't what it was")
	}
	r.Chunks.SetHome(first, last, false)
	if r.Chunks.IsHome(first) {
		t.Fatal("home couldn't be unmarked entirely")
	}
}

func TestHomeIsOnlyMarkedInsideTheColony(t *testing.T) {
	s, colony, _ := colonistAtWorkWithoutHome(t)
	first, last, _ := s.ECS.ColonyBounds(colony)
	outside := utils.Vec2i{X: first.X - 1, Y: first.Y}

	inputs := map[common.EntityId]ecs.InputState{s.Player: {HomeArea: ecs.HomeMarking{Mark: ecs.Home_Mark, First: outside, Last: first}}}
	if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
		t.Fatal(err)
	}
	if !s.Chunks.IsHome(first) || s.Chunks.IsHome(outside) {
		t.Fatal("marking home across the colony's edge didn't mark just the colony's side")
	}
	inputs[s.Player] = ecs.InputState{HomeArea: ecs.HomeMarking{Mark: ecs.Home_Unmark, First: first, Last: last}}
	if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, inputs); err != nil {
		t.Fatal(err)
	}
	if s.Chunks.IsHome(first) {
		t.Fatal("unmarking the whole colony left home")
	}
}

func colonistAtWorkWithoutHome(t *testing.T) (*testutil.Session, common.EntityId, common.EntityId) {
	t.Helper()
	s, colony, colonist := colonistAtWork(t)
	first, last, _ := s.ECS.ColonyBounds(colony)
	s.ECS.MarkHome(first, last, false)
	return s, colony, colonist
}

func TestBuildingOnHomeSpreadsItButBuildingElsewhereDoesnt(t *testing.T) {
	s, _, colonist := colonistAtWorkWithoutHome(t)
	onHome := groundAway(t, s, colonist, 6)
	s.ECS.MarkHome(onHome, onHome, true)
	s.Chunks.PlaceSite(onHome, data.Blueprint_Planks)
	if !live(t, s, 30*data.TPS, isBuilt(s, onHome)) {
		t.Fatal("the colonist didn't build the site on home")
	}

	r := data.HomeSpreadTiles
	for _, tile := range []utils.Vec2i{{X: onHome.X - r, Y: onHome.Y - r}, {X: onHome.X + r, Y: onHome.Y + r}} {
		if !s.Chunks.IsHome(tile) {
			t.Fatalf("%v, %d tiles from what was built on home, isn't home", tile, r)
		}
	}
	if s.Chunks.IsHome(utils.Vec2i{X: onHome.X + r + 1, Y: onHome.Y}) {
		t.Fatal("home spread further than it should")
	}

	// The player, as a builder, builds off home; the colonist leaves it alone
	for _, c := range []ecs.Component{
		ecs.NewInventoryComponent(2, 2),
		ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{}),
		ecs.NewConstructorComponent(3, 1, 250),
	} {
		s.ECS.AddComponent(s.Player, c)
	}
	elsewhere := groundAway(t, s, colonist, 3*r, onHome)
	if s.Chunks.IsHome(elsewhere) {
		t.Fatal("the ground chosen off home is home")
	}
	builderAt, found := utils.Vec2i{}, false
	for dy := -2; dy <= 2 && !found; dy++ {
		for dx := -2; dx <= 2 && !found; dx++ {
			builderAt = utils.Vec2i{X: elsewhere.X + dx, Y: elsewhere.Y + dy}
			_, sees := s.ECS.AimedTileInReach(utils.TilePosToWorldPos(builderAt), utils.TilePosToWorldPos(elsewhere), 3)
			found = builderAt != elsewhere && sees && s.Chunks.WalkSpeedPercent(builderAt) > 0
		}
	}
	if !found {
		t.Fatal("there's nowhere to build from next to the ground off home")
	}
	testutil.SetPos(t, s.ECS, s.Player, utils.TilePosToWorldPos(builderAt))
	s.Chunks.PlaceSite(elsewhere, data.Blueprint_Planks)
	s.Chunks.DeliverToSite(elsewhere, data.Item_Wood, planksCost(t))
	for range 10 {
		waitOutTheCooldown(s)
		construct(t, s, elsewhere, false)
	}
	if currentTile(s, elsewhere) != data.Tile_Planks || s.Chunks.IsHome(utils.Vec2i{X: elsewhere.X + 2, Y: elsewhere.Y}) {
		t.Fatal("building off home spread home around it")
	}
}
