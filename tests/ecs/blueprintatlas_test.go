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

func TestTheGamesBlueprintAtlasHasWoodenWalls(t *testing.T) {
	atlas, err := ecs.LoadBlueprintAtlasFromJson(filepath.Join(testutil.RepoRoot(), testutil.BlueprintsPath), testutil.ItemAtlas(t), testutil.Assets(t))
	if err != nil {
		t.Fatal(err)
	}

	def, ok := atlas.Def(data.Blueprint_WoodWall)
	if !ok || def.Name() != "Wooden wall" || def.Tile() != data.Tile_WoodWall || def.ImageAsset() != common.ImageAssetTag("tile_planks") || def.Stages() != 3 || def.Work() != 10 {
		t.Fatalf("the wooden wall blueprint is %+v (%v), want the wooden wall tile from the planks sheet's 3 stages, 10 work", def, ok)
	}
	if costs := def.Costs(); !slices.Equal(costs, []ecs.ItemCost{{Kind: data.Item_Wood, Amount: 2 * data.MilligramsPerKg}}) {
		t.Fatalf("wooden walls cost %v, want 2 kg of wood", costs)
	}
	if kinds := atlas.Kinds(); !slices.Equal(kinds, []data.BlueprintEnum{data.Blueprint_WoodWall, data.Blueprint_WoodDoor}) {
		t.Fatalf("the blueprints are %v, want wooden walls and doors", kinds)
	}

	door, _ := atlas.Def(data.Blueprint_WoodDoor)
	if ghost, frame := door.Ghost(); door.Feature() != data.Feature_WoodDoor || door.Tile() != data.Tile_None || ghost != "wood_door" || frame != 0 || door.Work() != 2*def.Work() {
		t.Fatalf("the wooden door blueprint is %+v, want the closed door, looking like it, taking twice a wall's work", door)
	}
	if costs := door.Costs(); !slices.Equal(costs, []ecs.ItemCost{{Kind: data.Item_Wood, Amount: 5 * data.MilligramsPerKg}}) {
		t.Fatalf("wooden doors cost %v, want 5 kg of wood", costs)
	}
	if ghost, frame := def.Ghost(); ghost != def.ImageAsset() || frame != 0 {
		t.Fatalf("the wooden wall looks like frame %d of %s before it's built, want the first frame of its own image", frame, ghost)
	}
}

func TestTheBlueprintAtlasRefusesBrokenBlueprints(t *testing.T) {
	for _, c := range []struct {
		name string
		json string
	}{
		{"the id meaning none", `[{"id": 0, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}], "work": 10}]`},
		{"no name", `[{"id": 1, "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}], "work": 10}]`},
		{"no work", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}], "work": 0}]`},
		{"no cost", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [], "work": 10}]`},
		{"an item that doesn't exist", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 99, "grams": 1000}], "work": 10}]`},
		{"none of an item", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 0}], "work": 10}]`},
		{"a count of what's weighed", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "count": 1}], "work": 10}]`},
		{"grams of what's counted", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 2, "grams": 200}], "work": 10}]`},
		{"an item twice", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}, {"kind": 1, "grams": 2000}], "work": 10}]`},
		{"both a tile and a feature", `[{"id": 1, "name": "A", "tile": 6, "feature": 4, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}], "work": 10}]`},
		{"neither a tile nor a feature", `[{"id": 1, "name": "A", "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}], "work": 10}]`},
		{"a ghost frame its image doesn't have", `[{"id": 1, "name": "A", "feature": 4, "imageAsset": "tile_planks", "ghost": {"asset": "wood_door", "frame": 4}, "costs": [{"kind": 1, "grams": 1000}], "work": 10}]`},
		{"an image that doesn't exist", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "nothing", "costs": [{"kind": 1, "grams": 1000}], "work": 10}]`},
		{"an image without stages", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "wood", "costs": [{"kind": 1, "grams": 1000}], "work": 10}]`},
		{"an id used twice", `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}], "work": 10},
			{"id": 1, "name": "B", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}], "work": 10}]`},
	} {
		if _, err := loadBlueprints(t, c.json); err == nil {
			t.Errorf("a blueprint with %s was accepted", c.name)
		}
	}
}

func TestBlueprintsAreListedInIdOrder(t *testing.T) {
	atlas, err := loadBlueprints(t, `[
		{"id": 3, "name": "C", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}], "work": 10},
		{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}], "work": 10},
		{"id": 2, "name": "B", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 1000}], "work": 10}]`)
	if err != nil {
		t.Fatal(err)
	}
	if kinds := atlas.Kinds(); !slices.Equal(kinds, []data.BlueprintEnum{1, 2, 3}) {
		t.Fatalf("listed %v, want 1, 2, 3", kinds)
	}
}

func TestASitesNeedsAreWhatsLeftOfItsCosts(t *testing.T) {
	atlas, err := loadBlueprints(t, `[{"id": 1, "name": "A", "tile": 6, "imageAsset": "tile_planks", "costs": [{"kind": 1, "grams": 3000}, {"kind": 2, "count": 1}], "work": 10}]`)
	if err != nil {
		t.Fatal(err)
	}
	def, _ := atlas.Def(1)

	site := ecs.ConstructionSite{Blueprint: 1, Delivered: []ecs.ItemCost{{Kind: data.Item_Wood, Amount: data.MilligramsPerKg}, {Kind: data.Item_Torch, Amount: 1}}}
	if site.Needs(data.Item_Wood, def) != 2*data.MilligramsPerKg || site.Needs(data.Item_Torch, def) != 0 || site.Supplied(def) {
		t.Fatalf("with 1 of 3 kg of wood and the torch delivered, it needs %d mg of wood and %d torches, supplied %v; want 2 kg, 0 and not supplied",
			site.Needs(data.Item_Wood, def), site.Needs(data.Item_Torch, def), site.Supplied(def))
	}
	site.Delivered[0].Amount = 3 * data.MilligramsPerKg
	if !site.Supplied(def) {
		t.Fatal("with everything delivered, the site isn't supplied")
	}
}
