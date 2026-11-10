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

func handleInput(t *testing.T, s *testutil.Session, e common.EntityId, in ecs.InputState) {
	t.Helper()
	if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, map[common.EntityId]ecs.InputState{e: in}); err != nil {
		t.Fatal(err)
	}
	testutil.RemoveScheduled(t, s.ECS)
}

// The colonist on the first tile of a row of open ground, holding its wood
func colonistHoldingWood(t *testing.T) (*testutil.Session, common.EntityId, utils.Vec2i) {
	t.Helper()
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, data.ItemReachTiles+3)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(row))
	handleInput(t, s, colonist, ecs.InputState{SelectSlot: 1, Inventory: ecs.InventoryAction{Kind: ecs.InventoryAction_Bind, From: 0, To: 0}})
	return s, colonist, row
}

func TestDroppingPutsAKilogramOfTheStackInHandOnTheAimedTileWithinReach(t *testing.T) {
	s, colonist, row := colonistHoldingWood(t)
	before := woodCarried(s, colonist)
	aimed := utils.Vec2i{X: row.X + data.ItemReachTiles, Y: row.Y}

	for range 2 {
		handleInput(t, s, colonist, ecs.InputState{DropItem: true, AimPos: utils.TilePosToWorldPos(aimed), HasAim: true})
	}
	if kind, n := onGround(s, aimed); kind != data.Item_Wood || n != 2*kg || woodCarried(s, colonist) != before-2*kg {
		t.Fatalf("dropping twice left %d mg of %v on the aimed tile and %d carried, want 2 kg of wood and %d", n, kind, woodCarried(s, colonist), before-2*kg)
	}
}

func TestDroppingAimedOutOfReachPutsItUnderfoot(t *testing.T) {
	s, colonist, row := colonistHoldingWood(t)
	far := utils.Vec2i{X: row.X + data.ItemReachTiles + 1, Y: row.Y}

	handleInput(t, s, colonist, ecs.InputState{DropItem: true, AimPos: utils.TilePosToWorldPos(far), HasAim: true})
	if _, ok := s.Chunks.GroundStack(far); ok {
		t.Fatal("wood was dropped on a tile out of reach")
	}
	if kind, n := onGround(s, row); kind != data.Item_Wood || n != 1*kg {
		t.Fatalf("the colonist's tile holds %d of %v, want the wood dropped there", n, kind)
	}
}

func TestDroppingAUniqueItemPutsAllOfItDownAndEmptiesTheHand(t *testing.T) {
	s, colonist, row := colonistHoldingWood(t)
	axe, _ := colonistAxe(s, colonist)
	inv, _ := s.ECS.Inventories.Get(colonist)
	handleInput(t, s, colonist, ecs.InputState{Inventory: ecs.InventoryAction{Kind: ecs.InventoryAction_Bind, From: inv.Capacity() - 1, To: 0}})
	if held, _ := inv.HotbarItem(0); held != axe {
		t.Fatal("the axe didn't go in hand")
	}

	aimed := utils.Vec2i{X: row.X + 1, Y: row.Y}
	handleInput(t, s, colonist, ecs.InputState{DropItem: true, AimPos: utils.TilePosToWorldPos(aimed), HasAim: true})
	if stack, ok := s.Chunks.GroundStack(aimed); !ok || stack.Unique == nil || stack.Unique.Root != axe {
		t.Fatal("the axe wasn't put down whole on the aimed tile")
	}
	if eqr, _ := s.ECS.Equippers.Get(colonist); len(eqr.Equipment()) != 0 || s.ECS.HasEntity(axe) {
		t.Fatal("the axe put down is still in hand, or still an entity")
	}
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestAKilledHolderLeavesWhatItCarriedWhereItDied(t *testing.T) {
	s, colonist, row := colonistHoldingWood(t)
	before := woodCarried(s, colonist)
	axe, _ := colonistAxe(s, colonist)

	s.ECS.ScheduleKillEntity(colonist)
	testutil.RemoveScheduled(t, s.ECS)
	wood, unique := 0, false
	for _, tile := range s.Chunks.GroundStacksIn(utils.Vec2i{X: row.X - 2, Y: row.Y - 2}, utils.Vec2i{X: row.X + 2, Y: row.Y + 2}) {
		stack, _ := s.Chunks.GroundStack(tile)
		if stack.Kind == data.Item_Wood {
			wood += stack.Amount
		}
		unique = unique || stack.Unique != nil && stack.Unique.Root == axe
	}
	if wood != before || !unique {
		t.Fatalf("the killed colonist left %d wood and its axe %v around where it died, want its %d and the axe", wood, unique, before)
	}
	if len(s.ECS.Items.GetEntities()) != 0 {
		t.Fatal("what the killed colonist carried is still around as entities")
	}
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestOnlyKilledEntitiesDropTheirLootAndTheirDropTablesAreSaved(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 1)
	table, err := ecs.NewDropTable(ecs.DropOutcome{ChancePercent: 100, Items: []ecs.ItemDrop{{Kind: data.Item_Wood, Amount: 3 * kg}}})
	if err != nil {
		t.Fatal(err)
	}
	pos := utils.TilePosToWorldPos(row)

	removed := testutil.AddMovable(t, s.ECS, pos, false, table)
	s.ECS.ScheduleRemoveEntity(removed)
	testutil.RemoveScheduled(t, s.ECS)
	if _, ok := s.Chunks.GroundStack(row); ok {
		t.Fatal("an entity removed without being killed dropped its loot")
	}

	killed := testutil.AddMovable(t, s.ECS, pos, false, table)
	members, err := s.ECS.GetCompositionComponents(killed)
	if err != nil {
		t.Fatal(err)
	}
	s.ECS.ScheduleRemoveEntity(killed)
	testutil.RemoveScheduled(t, s.ECS)
	for e, dtos := range members {
		comps := []ecs.Component{}
		for _, dto := range dtos {
			comp, err := ecs.DtoToComponent(dto)
			if err != nil {
				t.Fatal(err)
			}
			comps = append(comps, comp)
		}
		if err := s.ECS.AddEntityWithId(e, comps...); err != nil {
			t.Fatal(err)
		}
	}
	s.ECS.ScheduleKillEntity(killed)
	testutil.RemoveScheduled(t, s.ECS)
	if kind, n := onGround(s, row); kind != data.Item_Wood || n != 3*kg {
		t.Fatalf("a killed entity brought back from its saved components left %d of %v where it died, want 3 wood", n, kind)
	}
}

func TestDroppingLessThanAKilogramDropsAllOfIt(t *testing.T) {
	s, colonist, row := colonistHoldingWood(t)
	inv, _ := s.ECS.Inventories.Get(colonist)
	wood, _ := inv.HotbarItem(0)
	if err := s.ECS.UseUp(wood, woodCarried(s, colonist)-kg/2); err != nil {
		t.Fatal(err)
	}
	aimed := utils.Vec2i{X: row.X + 1, Y: row.Y}
	handleInput(t, s, colonist, ecs.InputState{DropItem: true, AimPos: utils.TilePosToWorldPos(aimed), HasAim: true})
	if _, n := onGround(s, aimed); n != kg/2 || s.ECS.HasEntity(wood) {
		t.Fatalf("dropping half a kilogram of wood put %d mg on the aimed tile, or left some in hand", n)
	}
}
