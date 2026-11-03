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
	if !x.Equipments.HasComponent(plank) {
		t.Fatal("a spawned plank can't be used in hand")
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
			if _, isEquipment := comp.(*ecs.Equipment); isEquipment {
				t.Fatal("a plank's use was saved, but it should come from the atlas")
			}
		}
	}

	// The atlas changed since the plank was saved, which the loaded plank goes by
	y := testutil.NewECS(t)
	atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t, `[{"id": 1, "name": "Planks", "imageAsset": "tree", "maxStack": 2}]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := y.SetItemAtlas(atlas); err != nil {
		t.Fatal(err)
	}
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
	if y.Equipments.HasComponent(plank) {
		t.Fatal("a loaded plank can still be used in hand, but the atlas says now it can't")
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
	if err := x.SetItemAtlas(atlas); err != nil {
		t.Fatal(err)
	}
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
	if err := x.SetItemAtlas(atlas); err != nil {
		t.Fatal(err)
	}
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

func TestItemUsesAreCheckedAgainstTheRegisteredAbilities(t *testing.T) {
	for _, c := range []struct {
		name string
		use  string
	}{
		{"an ability that doesn't exist", `{"ability": 999, "cooldownMs": 250, "params": {"rangeTiles": 3, "tile": 6}}`},
		{"no ability", `{"ability": 0, "cooldownMs": 250, "params": {}}`},
		{"an ability whose params the atlas can't give", `{"ability": 5, "cooldownMs": 250, "params": {}}`},
		{"params that don't fit the ability", `{"ability": 6, "cooldownMs": 250, "params": "planks"}`},
		{"no reach", `{"ability": 6, "cooldownMs": 250, "params": {"rangeTiles": 0, "tile": 6}}`},
		{"no reach for a feature", `{"ability": 8, "cooldownMs": 250, "params": {"rangeTiles": 0, "feature": 3}}`},
	} {
		atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t,
			`[{"id": 1, "name": "Planks", "imageAsset": "planks", "maxStack": 100, "use": `+c.use+`}]`))
		if err != nil {
			t.Fatal(err)
		}
		if err := testutil.NewECS(t).SetItemAtlas(atlas); err == nil {
			t.Errorf("an item using %s was accepted", c.name)
		}
	}

	if _, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t,
		`[{"id": 1, "name": "Planks", "imageAsset": "planks", "maxStack": 100, "use": {"ability": 6, "cooldownMs": -1, "params": {"rangeTiles": 3, "tile": 6}}}]`)); err == nil {
		t.Error("an item with a negative cooldown was accepted")
	}
}

func TestItemsCantPlaceTilesThatDontExist(t *testing.T) {
	atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t,
		`[{"id": 1, "name": "Planks", "imageAsset": "planks", "maxStack": 100, "use": {"ability": 6, "cooldownMs": 250, "params": {"rangeTiles": 3, "tile": 7}}}]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := testutil.NewECS(t).SetItemAtlas(atlas); err != nil {
		t.Fatal(err)
	}

	if err := atlas.CheckPlacedTiles(func(tile data.TileEnum) bool { return tile == 7 }); err != nil {
		t.Fatalf("an item placing a tile that exists was refused: %v", err)
	}
	if err := atlas.CheckPlacedTiles(func(tile data.TileEnum) bool { return tile != 7 }); err == nil {
		t.Fatal("an item placing a tile that doesn't exist was accepted")
	}
}

func TestItemsCantPlaceFeaturesThatDontExist(t *testing.T) {
	atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t,
		`[{"id": 1, "name": "Torch", "imageAsset": "planks", "maxStack": 100, "use": {"ability": 8, "cooldownMs": 250, "params": {"rangeTiles": 3, "feature": 5}}}]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := testutil.NewECS(t).SetItemAtlas(atlas); err != nil {
		t.Fatal(err)
	}

	if err := atlas.CheckPlacedFeatures(func(f data.FeatureEnum) bool { return f == 5 }); err != nil {
		t.Fatalf("an item placing a feature that exists was refused: %v", err)
	}
	if err := atlas.CheckPlacedFeatures(func(f data.FeatureEnum) bool { return f != 5 }); err == nil {
		t.Fatal("an item placing a feature that doesn't exist was accepted")
	}
	if err := atlas.CheckPlacedTiles(func(data.TileEnum) bool { return false }); err != nil {
		t.Fatalf("an item placing a feature was checked as placing a tile: %v", err)
	}
}

func TestStacksAreSpawnedUpToTheirLimit(t *testing.T) {
	x := testutil.NewECS(t)
	torches, err := x.SpawnItemStack(data.Item_Torch, 100, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	if item, _ := x.Items.Get(torches); item.Kind() != data.Item_Torch || item.Count() != 100 {
		t.Fatalf("spawned %v × %d, want 100 torches", item.Kind(), item.Count())
	}

	for _, count := range []int{0, -1, 101} {
		if _, err := x.SpawnItemStack(data.Item_Torch, count, utils.Vec2f{}); err == nil {
			t.Errorf("a stack of %d torches was spawned", count)
		}
	}
}
