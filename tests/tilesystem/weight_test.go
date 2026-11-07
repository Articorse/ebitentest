package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/tests/testutil"
	"testing"
)

func TestPickingUpStopsAtWhatTheCarrierCanStillCarry(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 1)
	start := s.ECS.CarriedWeight(colonist)
	if want := data.ColonistStartingWood*2 + 3; start != want {
		t.Fatalf("the colonist starts out carrying %d, want its wood and axe's %d", start, want)
	}
	s.ECS.LayDownKind(data.Item_Wood, 20, row)

	fits := (data.ColonistMaxCarryWeight - start) / 2
	if room := s.ECS.RoomFor(colonist, data.Item_Wood); room != fits {
		t.Fatalf("there's room for %d wood, want the %d it can still carry", room, fits)
	}
	if n, err := s.ECS.PickUpFromGround(colonist, row); err != nil || n != fits {
		t.Fatalf("picked up %d wood, want %d: %v", n, fits, err)
	}
	if _, n := onGround(s, row); n != 20-fits {
		t.Fatalf("%d wood was left, want %d", n, 20-fits)
	}

	s.Chunks.SetGroundStack(row, groundTorches(5))
	left := data.ColonistMaxCarryWeight - s.ECS.CarriedWeight(colonist)
	if n, _ := s.ECS.PickUpFromGround(colonist, row); n != left {
		t.Fatalf("picked up %d torches, want the %d that make up its limit", n, left)
	}
	if w := s.ECS.CarriedWeight(colonist); w != data.ColonistMaxCarryWeight {
		t.Fatalf("it carries %d, want exactly its limit of %d", w, data.ColonistMaxCarryWeight)
	}
	if n, _ := s.ECS.PickUpFromGround(colonist, row); n != 0 {
		t.Fatal("it picked up more at its limit")
	}

	over, _ := s.ECS.SpawnItemStack(data.Item_Wood, 5, testutil.WorldPos(t, s.ECS, colonist))
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
	if ok, err := s.ECS.LayDown(axe, row); err != nil || !ok {
		t.Fatalf("the axe wasn't laid down: %v", err)
	}
	inv, _ := s.ECS.Inventories.Get(colonist)
	wood, _ := inv.Slot(0)
	extra := (data.ColonistMaxCarryWeight - s.ECS.CarriedWeight(colonist)) / 2
	more := row
	more.X += 2
	s.Chunks.SetGroundStack(more, groundWood(extra))
	if n, _ := s.ECS.PickUpFromGround(colonist, more); n != extra {
		t.Fatalf("picked up %d of the %d wood that fill it up", n, extra)
	}

	// The axe weighs 3, so it doesn't fit at 38 but does at exactly 37
	if n, _ := s.ECS.PutDownOn(colonist, 0, 1, more); n != 1 {
		t.Fatal("couldn't put down a wood")
	}
	if n, _ := s.ECS.PickUpFromGround(colonist, row); n != 0 {
		t.Fatalf("the axe was picked up carrying %d with a limit of %d", s.ECS.CarriedWeight(colonist), data.ColonistMaxCarryWeight)
	}
	torch := row
	torch.X++
	s.Chunks.SetGroundStack(torch, groundTorches(1))
	if n, _ := s.ECS.PutDownOn(colonist, 0, 1, more); n != 1 {
		t.Fatal("couldn't put down another wood")
	}
	if n, _ := s.ECS.PickUpFromGround(colonist, torch); n != 1 {
		t.Fatal("couldn't pick up the torch")
	}
	if n, _ := s.ECS.PickUpFromGround(colonist, row); n != 1 {
		t.Fatalf("the axe wasn't picked up carrying %d with a limit of %d", s.ECS.CarriedWeight(colonist), data.ColonistMaxCarryWeight)
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
	if ok, err := s.ECS.LayDown(axe, row); err != nil || !ok {
		t.Fatalf("the axe wasn't laid down: %v", err)
	}
	carried := s.ECS.CarriedWeight(colonist)
	s.Commit()

	r := testutil.StartSession(t)
	if w := r.ECS.CarriedWeight(colonist); w != carried {
		t.Fatalf("after restarting the colonist carries %d, want %d", w, carried)
	}
	inv, _ := r.ECS.Inventories.Get(colonist)
	if inv.MaxWeight() != data.ColonistMaxCarryWeight {
		t.Fatalf("after restarting the colonist can carry %d, want %d", inv.MaxWeight(), data.ColonistMaxCarryWeight)
	}
	if n, _ := r.ECS.PickUpFromGround(colonist, row); n != 1 {
		t.Fatal("the axe couldn't be picked back up after restarting")
	}
	if w := r.ECS.CarriedWeight(colonist); w != carried+3 {
		t.Fatalf("with the axe back it carries %d, want %d", w, carried+3)
	}
}

func TestAColonistOnlyGathersWhatItCanCarry(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	a := groundAway(t, s, colonist, 5)
	harvest(s, a, a)
	looseWood(t, s, a, 10)
	fits := (data.ColonistMaxCarryWeight - s.ECS.CarriedWeight(colonist)) / 2
	before := woodCarried(s, colonist)

	live(t, s, 15*data.TPS, func() bool { return false })
	if woodCarried(s, colonist) != before+fits {
		t.Fatalf("the colonist gathered %d wood, want the %d it can carry", woodCarried(s, colonist)-before, fits)
	}
	if _, n := onGround(s, a); n != 10-fits || colonistState(s, colonist) != inputsources.Colonist_Idle {
		t.Fatalf("%d wood was left and the colonist is %v, want %d left and it idle", n, colonistState(s, colonist), 10-fits)
	}
}
