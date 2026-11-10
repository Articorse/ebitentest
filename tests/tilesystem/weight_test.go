package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func axeMass(t *testing.T, s *testutil.Session, colonist common.EntityId) int {
	t.Helper()
	axe, _ := colonistAxe(s, colonist)
	comp, ok := s.ECS.Items.Get(axe)
	if !ok {
		t.Fatal("the colonist's axe isn't an item")
	}
	return s.ECS.ItemMass(comp)
}

func TestPickingUpStopsAtWhatTheCarrierCanStillCarry(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 2)
	start := s.ECS.CarriedMass(colonist)
	if want := data.ColonistStartingWood + axeMass(t, s, colonist); start != want {
		t.Fatalf("the colonist starts out carrying %d, want its wood and axe's %d", start, want)
	}
	s.ECS.LayDownKind(data.Item_Wood, 20*kg, row)

	fits := data.ColonistMaxCarryMg - start
	if room := s.ECS.RoomFor(colonist, data.Item_Wood); room != fits {
		t.Fatalf("there's room for %d wood, want the %d it can still carry", room, fits)
	}
	if n, err := s.ECS.PickUpFromGround(colonist, row); err != nil || n != fits {
		t.Fatalf("picked up %d wood, want %d: %v", n, fits, err)
	}
	if _, n := onGround(s, row); n != 20*kg-fits {
		t.Fatalf("%d wood was left, want %d", n, 20*kg-fits)
	}

	if n, _ := s.ECS.PutDownOn(colonist, 0, 500*data.MilligramsPerGram, row); n != 500*data.MilligramsPerGram {
		t.Fatal("couldn't put down half a kilogram of wood")
	}
	torches := utils.Vec2i{X: row.X + 1, Y: row.Y}
	s.Chunks.SetGroundStack(torches, groundTorches(5))
	if n, _ := s.ECS.PickUpFromGround(colonist, torches); n != 2 {
		t.Fatalf("picked up %d torches of 200 g with half a kilogram to spare, want 2", n)
	}
	if n, _ := s.ECS.PickUpFromGround(colonist, torches); n != 0 {
		t.Fatalf("it picked up a torch with %d mg to spare", data.ColonistMaxCarryMg-s.ECS.CarriedMass(colonist))
	}

	over, _ := s.ECS.SpawnItemStack(data.Item_Wood, 5*kg, testutil.WorldPos(t, s.ECS, colonist))
	inv, _ := s.ECS.Inventories.Get(colonist)
	if err := s.ECS.PutInInventory(colonist, inv.FirstFreeSlot(), over); err != nil {
		t.Fatal(err)
	}
	if room := s.ECS.RoomFor(colonist, data.Item_Torch); room != 0 {
		t.Fatalf("over its limit, it has room for %d torches", room)
	}
	testutil.RemoveScheduled(t, s.ECS)
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestAUniqueItemIsOnlyPickedUpIfItsWeightStillFits(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 3)
	axe, _ := colonistAxe(s, colonist)
	axeMg := axeMass(t, s, colonist)
	if ok, err := s.ECS.LayDown(axe, row); err != nil || !ok {
		t.Fatalf("the axe wasn't laid down: %v", err)
	}
	inv, _ := s.ECS.Inventories.Get(colonist)
	wood, _ := inv.Slot(0)
	extra := data.ColonistMaxCarryMg - s.ECS.CarriedMass(colonist)
	more := row
	more.X += 2
	s.Chunks.SetGroundStack(more, groundWood(extra))
	if n, _ := s.ECS.PickUpFromGround(colonist, more); n != extra {
		t.Fatalf("picked up %d of the %d wood that fill it up", n, extra)
	}

	if n, _ := s.ECS.PutDownOn(colonist, 0, axeMg-1, more); n != axeMg-1 {
		t.Fatal("couldn't put down a milligram less wood than the axe weighs")
	}
	if n, _ := s.ECS.PickUpFromGround(colonist, row); n != 0 {
		t.Fatalf("the axe was picked up carrying %d with a limit of %d", s.ECS.CarriedMass(colonist), data.ColonistMaxCarryMg)
	}
	torchTile := row
	torchTile.X++
	s.Chunks.SetGroundStack(torchTile, groundTorches(1))
	torch, _ := testutil.ItemAtlas(t).Def(data.Item_Torch)
	if n, _ := s.ECS.PutDownOn(colonist, 0, torch.Mass(1)+1, more); n != torch.Mass(1)+1 {
		t.Fatal("couldn't put down a torch's weight and a milligram more wood")
	}
	if n, _ := s.ECS.PickUpFromGround(colonist, torchTile); n != 1 {
		t.Fatal("couldn't pick up the torch")
	}
	if n, _ := s.ECS.PickUpFromGround(colonist, row); n != 1 {
		t.Fatalf("the axe wasn't picked up carrying %d with a limit of %d", s.ECS.CarriedMass(colonist), data.ColonistMaxCarryMg)
	}
	if back, ok := colonistAxe(s, colonist); !ok || back != axe || countOf(s, wood) == 0 {
		t.Fatal("the axe isn't back with the colonist's wood")
	}
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestWeightsAndCarryLimitsAreSaved(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 2)
	axe, _ := colonistAxe(s, colonist)
	axeMg := axeMass(t, s, colonist)
	if ok, err := s.ECS.LayDown(axe, row); err != nil || !ok {
		t.Fatalf("the axe wasn't laid down: %v", err)
	}
	carried := s.ECS.CarriedMass(colonist)
	s.Commit()

	r := testutil.StartSession(t)
	if w := r.ECS.CarriedMass(colonist); w != carried {
		t.Fatalf("after restarting the colonist carries %d, want %d", w, carried)
	}
	inv, _ := r.ECS.Inventories.Get(colonist)
	if inv.MaxMass() != data.ColonistMaxCarryMg {
		t.Fatalf("after restarting the colonist can carry %d, want %d", inv.MaxMass(), data.ColonistMaxCarryMg)
	}
	if n, _ := r.ECS.PickUpFromGround(colonist, row); n != 1 {
		t.Fatal("the axe couldn't be picked back up after restarting")
	}
	if w := r.ECS.CarriedMass(colonist); w != carried+axeMg {
		t.Fatalf("with the axe back it carries %d, want %d", w, carried+axeMg)
	}
}

func TestAColonistOnlyGathersWhatItCanCarry(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	a := groundAway(t, s, colonist, 5)
	harvest(s, a, a)
	looseWood(t, s, a, 10*kg)
	fits := data.ColonistMaxCarryMg - s.ECS.CarriedMass(colonist)
	before := woodCarried(s, colonist)

	live(t, s, 15*data.TPS, func() bool { return false })
	if woodCarried(s, colonist) != before+fits {
		t.Fatalf("the colonist gathered %d wood, want the %d it can carry", woodCarried(s, colonist)-before, fits)
	}
	if _, n := onGround(s, a); n != 10*kg-fits || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatalf("%d wood was left and the colonist is %v, want %d left and it idle", n, colonistState(s, colonist), 10*kg-fits)
	}
}
