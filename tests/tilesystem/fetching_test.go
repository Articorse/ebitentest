package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

// The colonist's wood goes onto the storage tile, so that it has to fetch, and how much it was
func woodOntoStorage(t *testing.T, s *testutil.Session, colonist common.EntityId, tile utils.Vec2i) int {
	t.Helper()
	n := woodCarried(s, colonist)
	inv, _ := s.ECS.Inventories.Get(colonist)
	if put, err := s.ECS.PutDownOn(colonist, 0, n, tile); err != nil || put != n {
		t.Fatalf("put %d of the colonist's %d wood on storage: %v", put, n, err)
	}
	if _, ok := inv.Slot(0); ok || woodCarried(s, colonist) != 0 {
		t.Fatal("the colonist still has wood")
	}
	return n
}

func TestAColonistFetchesWhatASiteLacksFromStorageAndBuildsIt(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 8, 1, 1)
	markStorage(t, s, false, tile)
	before := woodOntoStorage(t, s, colonist, tile)
	site := groundAway(t, s, colonist, 6, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)

	if !live(t, s, 2*data.TPS, func() bool { return colonistState(s, colonist) == inputsources.Colonist_Fetch }) {
		t.Fatalf("the colonist didn't go to fetch, it's %v", colonistState(s, colonist))
	}
	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the colonist didn't build the blueprint, it's %v", colonistState(s, colonist))
	}
	if _, n := onGround(s, tile); n != before-wallCost(t) || woodCarried(s, colonist) != 0 {
		t.Fatalf("storage holds %d and the colonist %d wood, want just what the blueprint cost taken off %d", n, woodCarried(s, colonist), before)
	}
}

func TestAColonistFetchesWhatThereIsWhenStorageDoesntMakeUpAllItLacks(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 8, 1, 1)
	markStorage(t, s, false, tile)
	woodOntoStorage(t, s, colonist, tile)
	s.Chunks.SetGroundStack(tile, ecs.GroundStack{Kind: data.Item_Wood, Amount: wallCost(t) - 1})
	site := groundAway(t, s, colonist, 6, tile)
	s.Chunks.PlaceSite(site, data.Blueprint_WoodWall)

	def, _ := testutil.Blueprints(t).Def(data.Blueprint_WoodWall)
	delivered := func() bool { built, _ := s.Chunks.Site(site); return built.Needs(data.Item_Wood, def) == 1 }
	if !live(t, s, 20*data.TPS, delivered) {
		t.Fatalf("the colonist didn't deliver what there was in storage, it's %v", colonistState(s, colonist))
	}
	live(t, s, 3*data.TPS, func() bool { return false })
	if state := colonistState(s, colonist); state != inputsources.Colonist_Idle || woodCarried(s, colonist) != 0 {
		t.Fatalf("the colonist is %v carrying %d with nothing left in storage", state, woodCarried(s, colonist))
	}

	s.ECS.LayDownKind(data.Item_Wood, 1*kg, tile)
	if !live(t, s, 30*data.TPS, isBuilt(s, site)) {
		t.Fatalf("the colonist didn't build once the rest was in storage, it's %v", colonistState(s, colonist))
	}
}

func TestAColonistKeepsWhatSitesNeedWhenPuttingAway(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tile := storageSpot(t, s, colonist, 6, 1, 1)
	markStorage(t, s, false, tile)
	canBeWalledIn := func(t utils.Vec2i) bool {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				n := utils.Vec2i{X: t.X + dx, Y: t.Y + dy}
				if n != t && !s.Chunks.CanPlaceTile(n) && !s.Chunks.IsOpaqueTile(n) {
					return false
				}
			}
		}
		return true
	}
	walled := groundAwayWhere(t, s, colonist, 12, canBeWalledIn, tile)
	s.Chunks.PlaceSite(walled, data.Blueprint_WoodWall)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if wall := (utils.Vec2i{X: walled.X + dx, Y: walled.Y + dy}); wall != walled {
				s.Chunks.DamageTile(wall, 1000)
				if !s.Chunks.PlaceTile(wall, data.Tile_WoodWall) && !s.Chunks.IsOpaqueTile(wall) {
					t.Fatalf("couldn't wall in the blueprint at %v", wall)
				}
			}
		}
	}
	before := woodCarried(s, colonist)

	most := 0
	if !live(t, s, 20*data.TPS, func() bool {
		_, n := onGround(s, tile)
		most = max(most, n)
		return n == before-wallCost(t)
	}) {
		t.Fatal("the colonist didn't put away all but what the walled-in blueprint needs")
	}
	live(t, s, 3*data.TPS, func() bool { _, n := onGround(s, tile); most = max(most, n); return false })
	if most > before-wallCost(t) || woodCarried(s, colonist) != wallCost(t) {
		t.Fatalf("storage held up to %d and the colonist keeps %d, want %d kept", most, woodCarried(s, colonist), wallCost(t))
	}
}

func TestPickingUpSomeLeavesTheRest(t *testing.T) {
	s, colonist, row := colonistHoldingWood(t)
	near := utils.Vec2i{X: row.X + 1, Y: row.Y}
	s.ECS.LayDownKind(data.Item_Wood, 3*kg, near)
	before := woodCarried(s, colonist)

	handleInput(t, s, colonist, ecs.InputState{PickUp: ecs.ItemPickUp{PickUp: true, Tile: near, Amount: kg}})
	if _, n := onGround(s, near); n != 2*kg || woodCarried(s, colonist) != before+kg {
		t.Fatalf("picking up 1 kg of 3 left %d mg and gave %d", n, woodCarried(s, colonist)-before)
	}
	handleInput(t, s, colonist, ecs.InputState{PickUp: ecs.ItemPickUp{PickUp: true, Tile: near}})
	if _, ok := s.Chunks.GroundStack(near); ok || woodCarried(s, colonist) != before+3*kg {
		t.Fatal("picking up with no count didn't take the rest")
	}
}
