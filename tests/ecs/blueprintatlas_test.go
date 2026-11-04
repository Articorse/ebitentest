package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func loadBlueprints(t *testing.T, json string) (*ecs.BlueprintAtlas, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "blueprints.json")
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	return ecs.LoadBlueprintAtlasFromJson(path, testutil.ItemAtlas(t), testutil.Assets(t))
}

func TestTheGamesBlueprintAtlasHasPlanks(t *testing.T) {
	atlas, err := ecs.LoadBlueprintAtlasFromJson(filepath.Join(testutil.RepoRoot(), testutil.BlueprintsPath), testutil.ItemAtlas(t), testutil.Assets(t))
	if err != nil {
		t.Fatal(err)
	}

	def, ok := atlas.Def(data.Blueprint_Planks)
	if !ok || def.Name() != "Planks" || def.Tile() != data.Tile_Planks || def.ImageAsset() != common.ImageAssetTag("tile_planks") || def.Stages() != 3 || def.Work() != 10 {
		t.Fatalf("the planks blueprint is %+v (%v), want the planks tile from its sheet's 3 stages, 10 work", def, ok)
	}
	if costs := def.Costs(); !slices.Equal(costs, []ecs.ItemCost{{Kind: data.Item_Wood, Count: 1}}) {
		t.Fatalf("planks cost %v, want 1 wood", costs)
	}
	if kinds := atlas.Kinds(); !slices.Equal(kinds, []data.BlueprintEnum{data.Blueprint_Planks}) {
		t.Fatalf("the blueprints are %v, want only planks", kinds)
	}
}

func TestTheBlueprintAtlasRefusesBrokenBlueprints(t *testing.T) {
	for _, c := range []struct {
		name string
		json string
	}{
		{"the id meaning none", `[{"id": 0, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 10}]`},
		{"no name", `[{"id": 1, "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 10}]`},
		{"no work", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 0}]`},
		{"no cost", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [], "work": 10}]`},
		{"an item that doesn't exist", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 99, "count": 1}], "work": 10}]`},
		{"none of an item", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 0}], "work": 10}]`},
		{"an item twice", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}, {"kind": 1, "count": 2}], "work": 10}]`},
		{"an image that doesn't exist", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "nothing", "costs": [{"kind": 1, "count": 1}], "work": 10}]`},
		{"an image without stages", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "wood", "costs": [{"kind": 1, "count": 1}], "work": 10}]`},
		{"an id used twice", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 10},
			{"id": 1, "name": "B", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 10}]`},
	} {
		if _, err := loadBlueprints(t, c.json); err == nil {
			t.Errorf("a blueprint with %s was accepted", c.name)
		}
	}
}

func TestBlueprintsAreListedInIdOrder(t *testing.T) {
	atlas, err := loadBlueprints(t, `[
		{"id": 3, "name": "C", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 10},
		{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 10},
		{"id": 2, "name": "B", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 10}]`)
	if err != nil {
		t.Fatal(err)
	}
	if kinds := atlas.Kinds(); !slices.Equal(kinds, []data.BlueprintEnum{1, 2, 3}) {
		t.Fatalf("listed %v, want 1, 2, 3", kinds)
	}
}

func TestASitesNeedsAreWhatsLeftOfItsCosts(t *testing.T) {
	atlas, err := loadBlueprints(t, `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 3}, {"kind": 2, "count": 1}], "work": 10}]`)
	if err != nil {
		t.Fatal(err)
	}
	def, _ := atlas.Def(1)

	site := ecs.ConstructionSite{Blueprint: 1, Delivered: []ecs.ItemCost{{Kind: data.Item_Wood, Count: 1}, {Kind: data.Item_Torch, Count: 1}}}
	if site.Needs(data.Item_Wood, def) != 2 || site.Needs(data.Item_Torch, def) != 0 || site.Supplied(def) {
		t.Fatalf("with 1 of 3 wood and the torch delivered, it needs %d wood and %d torches, supplied %v; want 2, 0 and not supplied",
			site.Needs(data.Item_Wood, def), site.Needs(data.Item_Torch, def), site.Supplied(def))
	}
	site.Delivered[0].Count = 3
	if !site.Supplied(def) {
		t.Fatal("with everything delivered, the site isn't supplied")
	}
}
