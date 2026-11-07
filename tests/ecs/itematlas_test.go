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
		{"the unique items' id", `[{"id": 0, "name": "Nothing", "imageAsset": "wood", "maxStack": 1}]`},
		{"no name", `[{"id": 1, "imageAsset": "wood", "maxStack": 1}]`},
		{"no image", `[{"id": 1, "name": "Wood", "maxStack": 1}]`},
		{"a stack of nothing", `[{"id": 1, "name": "Wood", "imageAsset": "wood", "maxStack": 0}]`},
		{"a negative weight", `[{"id": 1, "name": "Wood", "imageAsset": "wood", "maxStack": 1, "weight": -1}]`},
		{"no light", `[{"id": 1, "name": "Torch", "imageAsset": "wood", "maxStack": 1, "light": {"level": 0, "rangeTiles": 10}}]`},
		{"light reaching nowhere", `[{"id": 1, "name": "Torch", "imageAsset": "wood", "maxStack": 1, "light": {"level": 150, "rangeTiles": 0}}]`},
		{"a frame that doesn't last", `[{"id": 1, "name": "Torch", "imageAsset": "wood", "maxStack": 1, "animation": [{"frame": 0, "durationMs": 0}]}]`},
		{"a negative frame", `[{"id": 1, "name": "Torch", "imageAsset": "wood", "maxStack": 1, "animation": [{"frame": -1, "durationMs": 150}]}]`},
		{"an id used twice", `[{"id": 1, "name": "A", "imageAsset": "wood", "maxStack": 1}, {"id": 1, "name": "B", "imageAsset": "wood", "maxStack": 1}]`},
	} {
		if _, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t, c.json)); err == nil {
			t.Errorf("an atlas with %s was accepted", c.name)
		}
	}
}

func TestTheGamesItemAtlasHasWood(t *testing.T) {
	atlas, err := ecs.LoadItemAtlasFromJson(filepath.Join(testutil.RepoRoot(), testutil.ItemAtlasPath))
	if err != nil {
		t.Fatal(err)
	}

	def, ok := atlas.Def(data.Item_Wood)
	if !ok || def.Name() != "Wood" || def.ImageAsset() != common.AssetImageWood || def.MaxStack() != 100 || def.Weight() != 2 {
		t.Fatalf("wood is %+v (%v), want Wood, drawn as wood, up to 100 a stack, weighing 2", def, ok)
	}
}

func TestGenericItemsAreOnlySavedAsTheirKindAndCount(t *testing.T) {
	x := testutil.NewECS(t)
	torch, err := x.SpawnItem(data.Item_Torch, utils.Vec2f{X: 3})
	if err != nil {
		t.Fatal(err)
	}
	if spr, ok := x.Sprites.Get(torch); !ok || spr.ImageAsset() != common.AssetSheetTorchPlaced {
		t.Fatal("a spawned torch isn't drawn as a torch")
	}
	if !x.Equipments.HasComponent(torch) {
		t.Fatal("a spawned torch can't be used in hand")
	}

	members, err := x.GetCompositionComponents(torch)
	if err != nil {
		t.Fatal(err)
	}
	for _, dto := range members[torch] {
		if comp, _ := ecs.DtoToComponent(dto); comp != nil {
			if _, isSprite := comp.(*ecs.Sprite); isSprite {
				t.Fatal("a torch's sprite was saved, but it should come from the atlas")
			}
			if _, isEquipment := comp.(*ecs.Equipment); isEquipment {
				t.Fatal("a torch's use was saved, but it should come from the atlas")
			}
		}
	}

	// The atlas changed since the torch was saved, which the loaded torch goes by
	y := testutil.NewECS(t)
	atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t, `[{"id": 2, "name": "Torch", "imageAsset": "tree", "maxStack": 2}]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := y.SetItemAtlas(atlas); err != nil {
		t.Fatal(err)
	}
	comps := []ecs.Component{}
	for _, dto := range members[torch] {
		comp, err := ecs.DtoToComponent(dto)
		if err != nil {
			t.Fatal(err)
		}
		comps = append(comps, comp)
	}
	if err := y.AddEntityWithId(torch, comps...); err != nil {
		t.Fatal(err)
	}

	if spr, ok := y.Sprites.Get(torch); !ok || spr.ImageAsset() != common.AssetImageTree {
		t.Fatal("a loaded torch isn't drawn the way the atlas says now")
	}
	if y.Equipments.HasComponent(torch) {
		t.Fatal("a loaded torch can still be used in hand, but the atlas says now it can't")
	}
	if pos := testutil.WorldPos(t, y, torch); pos != (utils.Vec2f{X: 3}) {
		t.Fatalf("the loaded torch is at %v, want where it was saved", pos)
	}
}

func TestStacksAreLimitedByTheAtlas(t *testing.T) {
	x := testutil.NewECS(t)
	atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t, `[{"id": 1, "name": "Wood", "imageAsset": "wood", "maxStack": 2}]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := x.SetItemAtlas(atlas); err != nil {
		t.Fatal(err)
	}
	owner := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewInventoryComponent(2, 2, 1000))

	for range 3 {
		wood, err := x.SpawnItem(data.Item_Wood, utils.Vec2f{})
		if err != nil {
			t.Fatal(err)
		}
		if err := x.GiveItem(owner, wood); err != nil {
			t.Fatalf("giving a wood: %v", err)
		}
		testutil.RemoveScheduled(t, x)
	}

	inv, _ := x.Inventories.Get(owner)
	first, _ := inv.Slot(0)
	second, _ := inv.Slot(1)
	firstItem, _ := x.Items.Get(first)
	secondItem, _ := x.Items.Get(second)
	if firstItem == nil || secondItem == nil || firstItem.Count() != 2 || secondItem.Count() != 1 {
		t.Fatal("3 wood stacking up to 2 didn't go into stacks of 2 and 1")
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
		{"id": 1, "name": "Wood", "imageAsset": "wood", "maxStack": 10},
		{"id": 2, "name": "Stones", "imageAsset": "wood", "maxStack": 10}]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := x.SetItemAtlas(atlas); err != nil {
		t.Fatal(err)
	}
	owner := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewInventoryComponent(1, 1, 1000))

	wood, err := x.SpawnItem(data.Item_Wood, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	if err := x.GiveItem(owner, wood); err != nil {
		t.Fatalf("giving a wood: %v", err)
	}

	if x.RoomFor(owner, 2) != 0 {
		t.Fatal("a stone can be stacked onto wood")
	}
}

func TestItemUsesAreCheckedAgainstTheRegisteredAbilities(t *testing.T) {
	for _, c := range []struct {
		name string
		use  string
	}{
		{"an ability that doesn't exist", `{"ability": 999, "cooldownMs": 250, "params": {"rangeTiles": 3, "feature": 3}}`},
		{"no ability", `{"ability": 0, "cooldownMs": 250, "params": {}}`},
		{"an ability whose params the atlas can't give", `{"ability": 5, "cooldownMs": 250, "params": {}}`},
		{"params that don't fit the ability", `{"ability": 8, "cooldownMs": 250, "params": "torch"}`},
		{"no reach", `{"ability": 8, "cooldownMs": 250, "params": {"rangeTiles": 0, "feature": 3}}`},
		{"the place tile ability, which is gone", `{"ability": 6, "cooldownMs": 250, "params": {"rangeTiles": 3, "tile": 6}}`},
	} {
		atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t,
			`[{"id": 1, "name": "Wood", "imageAsset": "wood", "maxStack": 100, "use": `+c.use+`}]`))
		if err != nil {
			t.Fatal(err)
		}
		if err := testutil.NewECS(t).SetItemAtlas(atlas); err == nil {
			t.Errorf("an item using %s was accepted", c.name)
		}
	}

	if _, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t,
		`[{"id": 1, "name": "Wood", "imageAsset": "wood", "maxStack": 100, "use": {"ability": 8, "cooldownMs": -1, "params": {"rangeTiles": 3, "feature": 3}}}]`)); err == nil {
		t.Error("an item with a negative cooldown was accepted")
	}
}

func TestItemsCantPlaceFeaturesThatDontExist(t *testing.T) {
	atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t,
		`[{"id": 1, "name": "Torch", "imageAsset": "wood", "maxStack": 100, "use": {"ability": 8, "cooldownMs": 250, "params": {"rangeTiles": 3, "feature": 5}}}]`))
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

func TestATorchsFlickerAndLightComeFromTheAtlasNotTheSave(t *testing.T) {
	x := testutil.NewECS(t)
	torch, err := x.SpawnItem(data.Item_Torch, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	if !x.Animations.HasComponent(torch) || !x.LightEmitters.HasComponent(torch) {
		t.Fatal("a spawned torch doesn't flicker or give light")
	}

	members, err := x.GetCompositionComponents(torch)
	if err != nil {
		t.Fatal(err)
	}
	comps := []ecs.Component{}
	for _, dto := range members[torch] {
		comp, err := ecs.DtoToComponent(dto)
		if err != nil {
			t.Fatal(err)
		}
		switch comp.(type) {
		case *ecs.Animation, *ecs.LightEmitter:
			t.Fatalf("a torch's %T was saved, but it should come from the atlas", comp)
		}
		comps = append(comps, comp)
	}

	y := testutil.NewECS(t)
	atlas, err := ecs.LoadItemAtlasFromJson(writeItemAtlas(t, `[{"id": 2, "name": "Torch", "imageAsset": "torch_placed", "maxStack": 100}]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := y.SetItemAtlas(atlas); err != nil {
		t.Fatal(err)
	}
	if err := y.AddEntityWithId(torch, comps...); err != nil {
		t.Fatal(err)
	}
	if y.Animations.HasComponent(torch) || y.LightEmitters.HasComponent(torch) {
		t.Fatal("a loaded torch flickers or gives light, but the atlas says now it doesn't")
	}
}
