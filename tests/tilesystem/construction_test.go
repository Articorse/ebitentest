package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func siteSprite(t *testing.T, s *testutil.Session, tile utils.Vec2i) (frame int, ghost bool) {
	t.Helper()
	center := utils.TilePosToWorldPos(tile)
	for _, sprite := range s.Chunks.FeatureSprites([2]utils.Vec2f{center, center}, 0) {
		if sprite.Pos == center {
			return sprite.Frame, sprite.Ghost
		}
	}
	t.Fatalf("nothing is drawn on %v", tile)
	return 0, false
}

func TestBlueprintsArePlacedOnlyOnBareGroundThatCanBeBuiltOn(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	grass := bareGrass(t, s)
	tree := findLoneTree(t, s)
	rock := tileOf(t, s, data.Tile_Rock)
	planks := findTile(t, s, func(tile utils.Vec2i) bool {
		_, hasFeature := s.Chunks.FeatureAt(tile)
		return tile != grass && currentTile(s, tile) == data.Tile_Grass && !hasFeature
	})
	if !s.Chunks.PlaceTile(planks, data.Tile_Planks) {
		t.Fatal("planks couldn't be placed to build next to")
	}

	for name, tile := range map[string]utils.Vec2i{"a tree": tree, "rock": rock, "planks": planks} {
		if s.Chunks.CanPlaceSite(tile) || s.Chunks.PlaceSite(tile, data.Blueprint_Planks) {
			t.Errorf("a blueprint was placed on %s", name)
		}
	}
	if s.Chunks.PlaceSite(grass, 99) {
		t.Error("a blueprint that isn't in the atlas was placed")
	}

	if !s.Chunks.CanPlaceSite(grass) || !s.Chunks.PlaceSite(grass, data.Blueprint_Planks) {
		t.Fatal("a blueprint wasn't placed on bare grass")
	}
	if s.Chunks.PlaceSite(grass, data.Blueprint_Planks) || s.Chunks.CanPlaceTile(grass) || s.Chunks.CanPlaceFeature(grass) {
		t.Fatal("something else can be placed on the blueprint")
	}

	center := utils.TilePosToWorldPos(grass)
	if s.Chunks.IsOpaqueTile(grass) || s.Chunks.IsBlockedAt(center) || s.Chunks.WalkSpeedPercent(grass) != 100 {
		t.Fatal("the blueprint can't be seen through or walked over")
	}
	if site, ok := s.Chunks.Site(grass); !ok || site.Blueprint != data.Blueprint_Planks || site.Health != 1 || site.MaxHealth != 100 {
		t.Fatalf("the placed site is %+v (%v), want the planks blueprint at 1 of 100 health", site, ok)
	}
	if frame, ghost := siteSprite(t, s, grass); frame != 0 || !ghost {
		t.Fatalf("a new site is drawn as frame %d, ghost %v; want a ghost of the finished tile", frame, ghost)
	}
}

func TestDirtAndRockFloorCanBeBuiltOnButWaterCant(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	if water := tileOf(t, s, data.Tile_WaterShallow); s.Chunks.CanPlaceSite(water) {
		t.Error("a blueprint can be placed in shallow water")
	}
	dirt := bareGrass(t, s)
	floor := tileOf(t, s, data.Tile_Rock)
	if ok, _ := s.Chunks.DamageTile(dirt, 1000); !ok {
		t.Fatal("the grass couldn't be dug up")
	}
	if ok, _ := s.Chunks.DamageTile(floor, 1000); !ok {
		t.Fatal("the rock couldn't be mined")
	}
	if currentTile(s, dirt) != data.Tile_Dirt || currentTile(s, floor) != data.Tile_RockFloor {
		t.Fatal("digging didn't leave dirt and rock floor")
	}

	for name, tile := range map[string]utils.Vec2i{"dirt": dirt, "rock floor": floor} {
		if !s.Chunks.CanPlaceSite(tile) {
			t.Errorf("a blueprint can't be placed on %s", name)
		}
	}
}

func TestASiteIsBuiltOnceSuppliedOverItsWorkAndBecomesTheTile(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tile := bareGrass(t, s)
	if !s.Chunks.PlaceSite(tile, data.Blueprint_Planks) {
		t.Fatal("the blueprint wasn't placed")
	}

	if worked, _ := s.Chunks.BuildSite(tile, 1); worked {
		t.Fatal("a site was worked on before its wood was delivered")
	}
	if taken := s.Chunks.DeliverToSite(tile, data.Item_Torch, 3); taken != 0 {
		t.Fatalf("the site took %d torches, which it doesn't need", taken)
	}
	if taken := s.Chunks.DeliverToSite(tile, data.Item_Wood, 5); taken != 1 {
		t.Fatalf("the site took %d of 5 wood, want the 1 it needs", taken)
	}
	if taken := s.Chunks.DeliverToSite(tile, data.Item_Wood, 5); taken != 0 {
		t.Fatalf("the supplied site took %d more wood", taken)
	}
	if site, _ := s.Chunks.Site(tile); len(site.Delivered) != 1 || site.Delivered[0] != (ecs.ItemCost{Kind: data.Item_Wood, Count: 1}) {
		t.Fatalf("the site has %v delivered, want 1 wood", site.Delivered)
	}

	// 10 work for planks' 100 health takes 10 strikes of 1, each adding 10 to the 1 it starts with
	wantFrames := []int{1, 1, 1, 2, 2, 2, 3, 3, 3}
	for i, want := range wantFrames {
		worked, built := s.Chunks.BuildSite(tile, 1)
		site, _ := s.Chunks.Site(tile)
		if !worked || built || site.Health != 1+10*(i+1) {
			t.Fatalf("strike %d worked %v, built %v, left %d health; want worked to %d", i+1, worked, built, site.Health, 1+10*(i+1))
		}
		if frame, ghost := siteSprite(t, s, tile); frame != want || ghost {
			t.Fatalf("after strike %d the site is drawn as frame %d, ghost %v; want stage %d", i+1, frame, ghost, want)
		}
	}

	if worked, built := s.Chunks.BuildSite(tile, 1); !worked || !built {
		t.Fatal("the tenth strike didn't build the planks")
	}
	if _, ok := s.Chunks.Site(tile); ok || currentTile(s, tile) != data.Tile_Planks {
		t.Fatal("the site wasn't replaced by planks")
	}
	if health, _ := s.Chunks.TileHealth(tile); health != 100 || !s.Chunks.IsBlockedAt(utils.TilePosToWorldPos(tile)) {
		t.Fatalf("the built planks have %d health, want a whole wall of 100", health)
	}
}

func TestMoreWorkBuildsInFewerStrikes(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tile := bareGrass(t, s)
	s.Chunks.PlaceSite(tile, data.Blueprint_Planks)
	s.Chunks.DeliverToSite(tile, data.Item_Wood, 1)

	if _, built := s.Chunks.BuildSite(tile, 5); built {
		t.Fatal("half the work built the planks")
	}
	if _, built := s.Chunks.BuildSite(tile, 5); !built {
		t.Fatal("two strikes of half the work didn't build the planks")
	}
}

func TestEveryStrikeAddsAtLeastOneHealth(t *testing.T) {
	testutil.TempWorld(t)
	path := filepath.Join(t.TempDir(), "blueprints.json")
	json := `[{"id": 1, "name": "Slow", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 1000}]`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	atlas, err := ecs.LoadBlueprintAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t))
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.StartSession(t)
	if err := s.Chunks.SetBlueprintAtlas(atlas); err != nil {
		t.Fatal(err)
	}
	tile := bareGrass(t, s)
	s.Chunks.PlaceSite(tile, 1)
	s.Chunks.DeliverToSite(tile, data.Item_Wood, 1)

	s.Chunks.BuildSite(tile, 1)
	if site, _ := s.Chunks.Site(tile); site.Health != 2 {
		t.Fatalf("a strike worth a tenth of a health point left %d health, want 2", site.Health)
	}
}

func TestSitesAreSavedWithTheirChunk(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tile := bareGrass(t, s)
	s.Chunks.PlaceSite(tile, data.Blueprint_Planks)
	s.Chunks.DeliverToSite(tile, data.Item_Wood, 1)
	for range 4 {
		s.Chunks.BuildSite(tile, 1)
	}

	s.Commit()
	r := testutil.StartSession(t)
	site, ok := r.Chunks.Site(tile)
	if !ok || site.Blueprint != data.Blueprint_Planks || site.Health != 41 || site.Delivered[0].Count != 1 {
		t.Fatalf("after restarting, the site is %+v (%v), want planks with its wood and 41 health", site, ok)
	}
	if frame, _ := siteSprite(t, r, tile); frame != 2 {
		t.Fatalf("after restarting, the site is drawn as frame %d, want stage 2", frame)
	}
}

func TestBlueprintsMustBuildATileWithHealthToBuildUp(t *testing.T) {
	for name, tile := range map[string]int{"a tile that doesn't exist": 99, "dirt, which has no health": 1} {
		testutil.TempWorld(t)
		path := filepath.Join(t.TempDir(), "blueprints.json")
		json := `[{"id": 1, "name": "A", "tile": ` + strconv.Itoa(tile) + `, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 10}]`
		if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
			t.Fatal(err)
		}
		atlas, err := ecs.LoadBlueprintAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t))
		if err != nil {
			t.Fatal(err)
		}
		s := testutil.StartSession(t)
		if err := s.Chunks.SetBlueprintAtlas(atlas); err == nil {
			t.Errorf("a blueprint building %s was accepted", name)
		}
	}
}
