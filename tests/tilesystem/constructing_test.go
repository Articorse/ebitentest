package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/inputsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A session whose player can build, holding the given wood, with a blueprint costing 3 wood placed on ground it can
// reach and see
func newBuilder(t *testing.T, wood int) (s *testutil.Session, site utils.Vec2i, stack common.EntityId) {
	t.Helper()
	testutil.TempWorld(t)
	path := filepath.Join(t.TempDir(), "blueprints.json")
	json := `[{"id": 1, "name": "Wall", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 3}], "work": 10}]`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	atlas, err := ecs.LoadBlueprintAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t))
	if err != nil {
		t.Fatal(err)
	}

	s = testutil.StartSession(t)
	s.ECS.SetBlueprintAtlas(atlas)
	if err := s.Chunks.SetBlueprintAtlas(atlas); err != nil {
		t.Fatal(err)
	}
	for _, c := range []ecs.Component{
		ecs.NewInventoryComponent(2, 2),
		ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{}),
		ecs.NewConstructorComponent(3, 1, 250),
	} {
		s.ECS.AddComponent(s.Player, c)
	}

	from := testutil.WorldPos(t, s.ECS, s.Player)
	origin := utils.WorldPosToTilePos(from)
	found := false
	for dy := -2; dy <= 2 && !found; dy++ {
		for dx := -2; dx <= 2 && !found; dx++ {
			site = utils.Vec2i{X: origin.X + dx, Y: origin.Y + dy}
			_, inReach := s.ECS.AimedTileInReach(from, utils.TilePosToWorldPos(site), 3)
			found = site != origin && inReach && s.Chunks.CanPlaceSite(site)
		}
	}
	if !found || !s.Chunks.PlaceSite(site, 1) {
		t.Fatal("there's no ground near the player to build on")
	}

	stack = -1
	if wood > 0 {
		if stack, err = s.ECS.SpawnItemStack(data.Item_Wood, wood, from); err != nil {
			t.Fatal(err)
		}
		if ok, err := s.ECS.PickUpItem(s.Player, stack); err != nil || !ok {
			t.Fatalf("picking up wood: %v, %v", ok, err)
		}
	}
	return s, site, stack
}

func construct(t *testing.T, s *testutil.Session, at utils.Vec2i, all bool) bool {
	t.Helper()
	handled, err := s.ECS.Construct(s.Player, utils.TilePosToWorldPos(at), all)
	if err != nil {
		t.Fatal(err)
	}
	testutil.RemoveScheduled(t, s.ECS)
	return handled
}

func delivered(s *testutil.Session, site utils.Vec2i) int {
	state, _ := s.Chunks.Site(site)
	if len(state.Delivered) == 0 {
		return -1
	}
	return state.Delivered[0].Count
}

func waitOutTheCooldown(s *testutil.Session) {
	s.ECS.TickIdx += 250 / data.TickMs
}

func TestAClickDeliversOneOfWhatsInHandAndShiftAllThatsNeeded(t *testing.T) {
	s, site, stack := newBuilder(t, 5)

	if !construct(t, s, site, false) || delivered(s, site) != 1 || countOf(s, stack) != 4 {
		t.Fatalf("a click delivered %d and left %d wood, want 1 delivered from the 5", delivered(s, site), countOf(s, stack))
	}
	if !construct(t, s, site, false) || delivered(s, site) != 1 {
		t.Fatal("a second click within the cooldown delivered more")
	}

	waitOutTheCooldown(s)
	if !construct(t, s, site, true) || delivered(s, site) != 3 || countOf(s, stack) != 2 {
		t.Fatalf("a shift click delivered up to %d and left %d wood, want the 2 still needed delivered and 2 left", delivered(s, site), countOf(s, stack))
	}

	waitOutTheCooldown(s)
	if construct(t, s, site, true) || countOf(s, stack) != 2 {
		t.Fatal("wood was delivered to a site that has all it needs, or the click didn't leave it to what's in hand")
	}
}

func TestDeliveringTheLastOfAStackEmptiesTheHand(t *testing.T) {
	s, site, stack := newBuilder(t, 2)

	construct(t, s, site, true)
	if delivered(s, site) != 2 || s.ECS.HasEntity(stack) {
		t.Fatal("delivering the whole stack didn't use it up")
	}
	inv, _ := s.ECS.Inventories.Get(s.Player)
	if _, ok := inv.HotbarItem(0); ok {
		t.Fatal("the used up stack is still on the hotbar")
	}
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestAnEmptyHandBuildsASuppliedSiteStrikeByStrike(t *testing.T) {
	s, site, _ := newBuilder(t, 3)
	if construct(t, s, site, true); delivered(s, site) != 3 {
		t.Fatal("the wood wasn't delivered")
	}
	waitOutTheCooldown(s)

	for strike := 1; strike <= 9; strike++ {
		if !construct(t, s, site, false) {
			t.Fatalf("strike %d wasn't taken by the site", strike)
		}
		if state, _ := s.Chunks.Site(site); state.Health != 1+10*strike {
			t.Fatalf("after strike %d the site has %d health, want %d", strike, state.Health, 1+10*strike)
		}
		if construct(t, s, site, false); func() bool { state, _ := s.Chunks.Site(site); return state.Health != 1+10*strike }() {
			t.Fatalf("a strike within the cooldown after strike %d worked on the site", strike)
		}
		waitOutTheCooldown(s)
	}

	construct(t, s, site, false)
	if _, ok := s.Chunks.Site(site); ok || currentTile(s, site) != data.Tile_Planks {
		t.Fatal("the tenth strike didn't build the wall")
	}
}

func TestAnEmptyHandDoesNothingToASiteMissingMaterials(t *testing.T) {
	s, site, _ := newBuilder(t, 0)
	if construct(t, s, site, false) {
		t.Fatal("an empty hand was taken by a site still waiting for its wood")
	}
	if state, _ := s.Chunks.Site(site); state.Health != 1 {
		t.Fatal("a site was worked on before its wood was delivered")
	}
}

func TestTheLastStrikeWaitsForTheTileToBeClear(t *testing.T) {
	s, site, _ := newBuilder(t, 3)
	construct(t, s, site, true)
	for range 9 {
		waitOutTheCooldown(s)
		construct(t, s, site, false)
	}

	shape, err := shapes.NewRectangleShape(6, 6, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	blocker := testutil.AddMovable(t, s.ECS, utils.TilePosToWorldPos(site), false, ecs.NewPhysicsColliderComponent(1, 1, ecs.Collider_Mob, shape))
	waitOutTheCooldown(s)
	if !construct(t, s, site, false) || currentTile(s, site) == data.Tile_Planks {
		t.Fatal("the wall was built on top of something standing in it")
	}
	if state, _ := s.Chunks.Site(site); state.Health != 91 {
		t.Fatalf("the blocked strike left %d health, want the 91 it had", state.Health)
	}

	s.ECS.ScheduleRemoveEntity(blocker)
	testutil.RemoveScheduled(t, s.ECS)
	if !construct(t, s, site, false) || currentTile(s, site) != data.Tile_Planks {
		t.Fatal("the wall wasn't built once the tile was clear, right away since the blocked strike didn't start a cooldown")
	}
}

func TestSitesOutOfReachAreLeftToWhatsInHand(t *testing.T) {
	s, _, stack := newBuilder(t, 3)
	from := utils.WorldPosToTilePos(testutil.WorldPos(t, s.ECS, s.Player))
	far := findTile(t, s, func(tile utils.Vec2i) bool {
		d := utils.Dist2(tile, from)
		return d > 3*3 && d <= 5*5 && s.Chunks.CanPlaceSite(tile)
	})
	s.Chunks.PlaceSite(far, 1)

	if construct(t, s, far, true) || countOf(s, stack) != 3 {
		t.Fatal("wood was delivered to a site 4 tiles away")
	}
}

func TestAClickOnASiteInputsConstruction(t *testing.T) {
	s, site, stack := newBuilder(t, 3)
	click := ecs.InputState{MainHandEqAbility1: 1, AimPos: utils.TilePosToWorldPos(site), HasAim: true, All: true}
	if aim, ok := s.ECS.Aims.Get(s.Player); ok {
		aim.SetTarget(click.AimPos)
	} else {
		s.ECS.AddComponent(s.Player, ecs.NewAimComponent())
	}
	if err := inputsystem.HandleInputs(utils.Vec2f{}, s.ECS, map[common.EntityId]ecs.InputState{s.Player: click}); err != nil {
		t.Fatal(err)
	}
	if delivered(s, site) != 3 || s.ECS.HasEntity(stack) && countOf(s, stack) != 0 {
		t.Fatalf("a shift click delivered %d wood, want all 3", delivered(s, site))
	}
}

func TestTheConstructorsCooldownIsSaved(t *testing.T) {
	s, site, _ := newBuilder(t, 5)
	construct(t, s, site, false)

	members, err := s.ECS.GetCompositionComponents(s.Player)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, map[common.EntityId][]ecs.ComponentDto{s.Player: members[s.Player]})
	restored, err := y.GetCompositionComponents(s.Player)
	if err != nil {
		t.Fatal(err)
	}
	constructorOf := func(dtos []ecs.ComponentDto) string {
		for _, d := range dtos {
			if fmt.Sprintf("%T", d) == "*ecs.constructorDto" {
				return fmt.Sprintf("%+v", d)
			}
		}
		return ""
	}
	if before, after := constructorOf(members[s.Player]), constructorOf(restored[s.Player]); before == "" || before != after {
		t.Fatalf("the constructor was saved as %s and came back as %s", before, after)
	}
	if !strings.Contains(constructorOf(members[s.Player]), fmt.Sprintf("ReadyTick:%d", s.ECS.TickIdx+250/data.TickMs)) {
		t.Fatalf("the saved constructor %s isn't waiting out the cooldown of the delivery", constructorOf(members[s.Player]))
	}
}

func countOf(s *testutil.Session, item common.EntityId) int {
	itemComp, ok := s.ECS.Items.Get(item)
	if !ok {
		return 0
	}
	return itemComp.Count()
}
