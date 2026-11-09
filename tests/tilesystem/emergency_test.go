package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

// Another colonist off to fell a tree, held inside a room until it's told to get out of the way of the room's last
// wall, which the colonist at work builds
func choppingWhenTheWallsFinished(
	t *testing.T,
	treeAt func(s *testutil.Session, builder common.EntityId, middle utils.Vec2i) utils.Vec2i,
) (s *testutil.Session, builder, other common.EntityId, tree, gap utils.Vec2i, inside func(utils.Vec2i) bool) {
	t.Helper()
	s, colony, builder := colonistAtWork(t)
	middle := doorYard(t, s, builder)
	gap = at(middle, 0, 3)
	inside = roomWithAWallToFinish(t, s, middle, gap)
	live(t, s, 1, func() bool { return false })
	tree = treeAt(s, builder, middle)
	if !s.Chunks.PlaceFeature(tree, data.Feature_Tree) {
		t.Fatalf("couldn't plant a tree on %v", tree)
	}
	harvest(s, tree)
	other = anotherColonist(t, s, colony, builder)
	if !live(t, s, 10*data.TPS, func() bool {
		testutil.SetPos(t, s.ECS, other, utils.TilePosToWorldPos(middle))
		return colonistJob(s, other).Emergency.Kind == inputsources.Emergency_GetOut
	}) {
		t.Fatal("the other colonist wasn't told to get out of the way")
	}
	if !inputsources.InEmergency(other, s.ECS) || inputsources.InEmergency(builder, s.ECS) {
		t.Fatal("the other colonist isn't shown in an emergency, or the builder is")
	}
	if paused := colonistJob(s, other).Paused; paused == nil || paused.State != inputsources.Colonist_Chop || paused.Stops[0].Tile != tree {
		t.Fatalf("the other colonist's paused job is %+v, want felling the tree", paused)
	}
	if !live(t, s, 30*data.TPS, isBuilt(s, gap)) {
		t.Fatal("the wall wasn't finished")
	}
	if inside(colonistTile(t, s, other)) {
		t.Fatal("the other colonist was shut in")
	}
	live(t, s, 1, func() bool { return false })
	if inputsources.InEmergency(other, s.ECS) {
		t.Fatal("the other colonist is still shown in an emergency once the wall's built")
	}
	return s, builder, other, tree, gap, inside
}

func TestAColonistCarriesOnWithItsJobAfterAnEmergency(t *testing.T) {
	s, builder, other, tree, _, _ := choppingWhenTheWallsFinished(t, func(s *testutil.Session, builder common.EntityId, middle utils.Vec2i) utils.Vec2i {
		return groundAwayWhere(t, s, builder, 6, func(tile utils.Vec2i) bool { return utils.Dist2(tile, middle) > 6*6 })
	})
	if job := colonistJob(s, other); job.State != inputsources.Colonist_Chop || job.Stops[0].Tile != tree || job.Paused != nil {
		t.Fatalf("after the emergency the other colonist is on %v, want felling the tree again", job.Stops)
	}
	if job := colonistJob(s, builder); job.State == inputsources.Colonist_Chop && job.Stops[0].Tile == tree {
		t.Fatal("the builder took over felling the tree while it was claimed")
	}
}

func TestAColonistDropsAJobItCantGetToAfterAnEmergency(t *testing.T) {
	s, _, other, tree, _, _ := choppingWhenTheWallsFinished(t, func(_ *testutil.Session, _ common.EntityId, middle utils.Vec2i) utils.Vec2i {
		return at(middle, -2, -2)
	})
	if job := colonistJob(s, other); job.State == inputsources.Colonist_Chop && job.Stops[0].Tile == tree {
		t.Fatal("the other colonist carried on felling the tree it was shut out from")
	}
	if colonistJob(s, other).Paused != nil {
		t.Fatal("the other colonist's job is still paused")
	}
}

func TestAnEmergencyIsOverInTime(t *testing.T) {
	s, colony, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	gap := at(middle, 0, 3)
	roomWithAWallToFinish(t, s, middle, gap)
	live(t, s, 1, func() bool { return false })
	other := anotherColonist(t, s, colony, colonist)
	if !live(t, s, 10*data.TPS, func() bool {
		testutil.SetPos(t, s.ECS, other, utils.TilePosToWorldPos(middle))
		return colonistJob(s, other).Emergency.Kind == inputsources.Emergency_GetOut
	}) {
		t.Fatal("the other colonist wasn't told to get out of the way")
	}
	s.ECS.SetDisabled(colonist, true)
	until := colonistJob(s, other).Emergency.Until
	if !live(t, s, int(data.ColonistGetOutTicks)+2, func() bool { return colonistJob(s, other).Emergency.Kind == inputsources.Emergency_None }) {
		t.Fatal("the emergency didn't end in time")
	}
	if s.ECS.TickIdx < until {
		t.Fatalf("the emergency ended on tick %d, before %d", s.ECS.TickIdx, until)
	}
}

func TestAnEmergencyAndTheJobItPausedAreSaved(t *testing.T) {
	s, colony, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	roomWithAWallToFinish(t, s, middle, at(middle, 0, 3))
	live(t, s, 1, func() bool { return false })
	other := anotherColonist(t, s, colony, colonist)
	tree := groundAwayWhere(t, s, colonist, 6, func(tile utils.Vec2i) bool { return utils.Dist2(tile, middle) > 6*6 })
	if !s.Chunks.PlaceFeature(tree, data.Feature_Tree) {
		t.Fatalf("couldn't plant a tree on %v", tree)
	}
	harvest(s, tree)
	if !live(t, s, 10*data.TPS, func() bool {
		testutil.SetPos(t, s.ECS, other, utils.TilePosToWorldPos(middle))
		return colonistJob(s, other).Paused != nil
	}) {
		t.Fatal("the other colonist's job wasn't paused")
	}
	want := *colonistJob(s, other)

	s.Commit()
	r := testutil.StartSession(t)
	got := colonistJob(r, other)
	if got.Emergency != want.Emergency || got.Paused == nil || got.Paused.State != want.Paused.State || got.Paused.ClaimUntil != want.Paused.ClaimUntil {
		t.Fatalf("after restarting the emergency is %+v with %+v paused, want %+v with %+v", got.Emergency, got.Paused, want.Emergency, want.Paused)
	}
}
