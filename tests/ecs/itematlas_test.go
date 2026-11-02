package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"os"
	"path/filepath"
	"testing"
)

func writeItemAtlas(t *testing.T, json string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "items.json")
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTheItemAtlasRefusesBrokenItems(t *testing.T) {
	for _, c := range []struct {
		name string
		json string
	}{
		{"the unique items' id", `[{"id": 0, "name": "Nothing", "imageAsset": "planks", "maxStack": 1}]`},
		{"no name", `[{"id": 1, "imageAsset": "planks", "maxStack": 1}]`},
		{"no image", `[{"id": 1, "name": "Planks", "maxStack": 1}]`},
		{"a stack of nothing", `[{"id": 1, "name": "Planks", "imageAsset": "planks", "maxStack": 0}]`},
		{"an id used twice", `[{"id": 1, "name": "A", "imageAsset": "planks", "maxStack": 1}, {"id": 1, "name": "B", "imageAsset": "planks", "maxStack": 1}]`},
	} {
		if _, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t, c.json)); err == nil {
			t.Errorf("an atlas with %s was accepted", c.name)
		}
	}
}

func TestTheGamesItemAtlasHasPlanks(t *testing.T) {
	atlas, err := ecs.LoadItemAtlasFromJson(filepath.Join(testutil.RepoRoot(), testutil.ItemAtlasPath))
	if err != nil {
		t.Fatal(err)
	}

	def, ok := atlas.Def(data.Item_Planks)
	if !ok || def.Name() != "Planks" || def.ImageAsset() != common.AssetImagePlanks || def.MaxStack() != 100 {
		t.Fatalf("planks are %+v (%v), want Planks, drawn as planks, up to 100 a stack", def, ok)
	}
}

func TestGenericItemsAreOnlySavedAsTheirKindAndCount(t *testing.T) {
	x := testutil.NewECS(t)
	plank, err := x.SpawnItem(data.Item_Planks, utils.Vec2f{X: 3})
	if err != nil {
		t.Fatal(err)
	}
	if spr, ok := x.Sprites.Get(plank); !ok || spr.ImageAsset() != common.AssetImagePlanks {
		t.Fatal("a spawned plank isn't drawn as planks")
	}

	members, err := x.GetCompositionComponents(plank)
	if err != nil {
		t.Fatal(err)
	}
	for _, dto := range members[plank] {
		if comp, _ := ecs.DtoToComponent(dto); comp != nil {
			if _, isSprite := comp.(*ecs.Sprite); isSprite {
				t.Fatal("a plank's sprite was saved, but it should come from the atlas")
			}
		}
	}

	// The atlas changed since the plank was saved, which the loaded plank goes by
	y := testutil.NewECS(t)
	atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t, `[{"id": 1, "name": "Planks", "imageAsset": "tree", "maxStack": 2}]`))
	if err != nil {
		t.Fatal(err)
	}
	y.SetItemAtlas(atlas)
	comps := []ecs.Component{}
	for _, dto := range members[plank] {
		comp, err := ecs.DtoToComponent(dto)
		if err != nil {
			t.Fatal(err)
		}
		comps = append(comps, comp)
	}
	if err := y.AddEntityWithId(plank, comps...); err != nil {
		t.Fatal(err)
	}

	if spr, ok := y.Sprites.Get(plank); !ok || spr.ImageAsset() != common.AssetImageTree {
		t.Fatal("a loaded plank isn't drawn the way the atlas says now")
	}
	if pos := testutil.WorldPos(t, y, plank); pos != (utils.Vec2f{X: 3}) {
		t.Fatalf("the loaded plank is at %v, want where it was saved", pos)
	}
}

func TestStacksAreLimitedByTheAtlas(t *testing.T) {
	x := testutil.NewECS(t)
	atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t, `[{"id": 1, "name": "Planks", "imageAsset": "planks", "maxStack": 2}]`))
	if err != nil {
		t.Fatal(err)
	}
	x.SetItemAtlas(atlas)
	owner := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewInventoryComponent(2))

	for range 3 {
		plank, err := x.SpawnItem(data.Item_Planks, utils.Vec2f{})
		if err != nil {
			t.Fatal(err)
		}
		if ok, err := x.PickUpItem(owner, plank); err != nil || !ok {
			t.Fatalf("picking up a plank: %v, %v", ok, err)
		}
		testutil.RemoveScheduled(t, x)
	}

	inv, _ := x.Inventories.Get(owner)
	first, _ := inv.Slot(0)
	second, _ := inv.Slot(1)
	firstItem, _ := x.Items.Get(first)
	secondItem, _ := x.Items.Get(second)
	if firstItem == nil || secondItem == nil || firstItem.Count() != 2 || secondItem.Count() != 1 {
		t.Fatal("3 planks stacking up to 2 didn't go into stacks of 2 and 1")
	}
}

func TestOnlyItemsInTheAtlasCanBeSpawned(t *testing.T) {
	x := testutil.NewECS(t)
	for _, kind := range []data.ItemEnum{data.Item_None, 999} {
		if _, err := x.SpawnItem(kind, utils.Vec2f{}); err == nil {
			t.Errorf("item %v was spawned, but isn't in the atlas", kind)
		}
	}
}

func TestOnlyItemsOfTheSameKindStack(t *testing.T) {
	x := testutil.NewECS(t)
	atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t, `[
		{"id": 1, "name": "Planks", "imageAsset": "planks", "maxStack": 10},
		{"id": 2, "name": "Stones", "imageAsset": "planks", "maxStack": 10}]`))
	if err != nil {
		t.Fatal(err)
	}
	x.SetItemAtlas(atlas)
	owner := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewInventoryComponent(1))

	plank, err := x.SpawnItem(data.Item_Planks, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := x.PickUpItem(owner, plank); err != nil || !ok {
		t.Fatalf("picking up a plank: %v, %v", ok, err)
	}

	stone, err := x.SpawnItem(2, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	if x.CanPickUp(owner, stone) {
		t.Fatal("a stone can be stacked onto planks")
	}
}
