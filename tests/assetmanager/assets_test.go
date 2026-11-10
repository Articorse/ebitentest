package assetmanager_test

import (
	"ebittest/assetmanager"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// An asset list in a folder of its own, next to a copy of the game's sprites and debug image
func writeAssetList(t *testing.T, list string) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"sprites", "debug", "tiles"} {
		if err := os.CopyFS(filepath.Join(dir, sub), os.DirFS(filepath.Join(testutil.RepoRoot(), "assets", sub))); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "assets.json")
	if err := os.WriteFile(path, []byte(list), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const debugEntry = `{"tag": "debug_16x16", "path": "debug/16x16.png", "frameCount": 1, "frameSize": {"x": 16, "y": 16}}`

func TestTheGamesAssetsHaveEveryTagCodeUses(t *testing.T) {
	assets := testutil.Assets(t)
	if err := assets.CheckTags(common.CodeAssetTags); err != nil {
		t.Fatal(err)
	}
	if err := assets.CheckTags([]common.ImageAssetTag{"nothing"}); err == nil {
		t.Fatal("a tag that isn't in the list was accepted")
	}
	if err := testutil.ItemAtlas(t).CheckImageAssets(assets); err != nil {
		t.Fatal(err)
	}

	slime, err := assets.GetAsset(common.AssetSheetSlime)
	if err != nil || len(slime.Frames) != 3 || slime.Frames[1].Bounds().Dx() != 32 {
		t.Fatalf("the slime has %d frames (%v), want 3 of 32 pixels", len(slime.Frames), err)
	}
}

func TestPathsAreRelativeToTheAssetList(t *testing.T) {
	path := writeAssetList(t, `[`+debugEntry+`, {"tag": "tree", "path": "sprites/tree.png", "frameCount": 1, "frameSize": {"x": 32, "y": 32}}]`)
	t.Chdir(t.TempDir())
	assets, err := assetmanager.LoadAssetManagerFromJson(path)
	if err != nil {
		t.Fatal(err)
	}
	if tree, _ := assets.GetAsset(common.AssetImageTree); tree.Frames[0].Bounds().Dx() != 32 {
		t.Fatal("the tree wasn't loaded from beside the asset list")
	}
}

func TestASheetsFramesGoAlongEachRowInTurn(t *testing.T) {
	for _, count := range []int{16, 13} {
		assets, err := assetmanager.LoadAssetManagerFromJson(writeAssetList(t, `[`+debugEntry+`, {"tag": "wall", "path": "tiles/wood_wall_ss.png", "frameCount": `+strconv.Itoa(count)+`, "frameSize": {"x": 16, "y": 16}}]`))
		if err != nil {
			t.Fatal(err)
		}
		wall, _ := assets.GetAsset("wall")
		if len(wall.Frames) != count || wall.Frames[6].Bounds() != image.Rect(32, 16, 48, 32) || wall.Frames[count-1].Bounds().Min != image.Pt((count-1)%4*16, (count-1)/4*16) {
			t.Fatalf("a 4×4 sheet of %d frames has frames 6 and %d at %v and %v", count, count-1, wall.Frames[6].Bounds(), wall.Frames[count-1].Bounds())
		}
	}
}

func TestAMissingImageIsDrawnAsTheDebugImage(t *testing.T) {
	assets, err := assetmanager.LoadAssetManagerFromJson(writeAssetList(t,
		`[`+debugEntry+`, {"tag": "gone", "path": "sprites/gone.png", "frameCount": 3, "frameSize": {"x": 32, "y": 32}}]`))
	if err != nil {
		t.Fatal(err)
	}
	gone, _ := assets.GetAsset("gone")
	debug, _ := assets.GetAsset(common.AssetDebug16x16)
	if len(gone.Frames) != 3 || gone.Frames[2] != debug.Frames[0] {
		t.Fatal("a missing sheet of 3 frames isn't 3 frames of the debug image")
	}
}

func TestTheAssetListRefusesBrokenAssets(t *testing.T) {
	for _, c := range []struct {
		name string
		list string
	}{
		{"no debug image", `[{"tag": "tree", "path": "sprites/tree.png", "frameCount": 1, "frameSize": {"x": 32, "y": 32}}]`},
		{"a tag used twice", `[` + debugEntry + `, ` + debugEntry + `]`},
		{"no tag", `[` + debugEntry + `, {"tag": "", "path": "sprites/tree.png", "frameCount": 1, "frameSize": {"x": 32, "y": 32}}]`},
		{"no frames", `[` + debugEntry + `, {"tag": "tree", "path": "sprites/tree.png", "frameCount": 0, "frameSize": {"x": 32, "y": 32}}]`},
		{"more frames than the image has", `[` + debugEntry + `, {"tag": "tree", "path": "sprites/tree.png", "frameCount": 2, "frameSize": {"x": 32, "y": 32}}]`},
		{"frames shorter than the image", `[` + debugEntry + `, {"tag": "tree", "path": "sprites/tree.png", "frameCount": 2, "frameSize": {"x": 16, "y": 16}}]`},
		{"an empty row of frames", `[` + debugEntry + `, {"tag": "wall", "path": "tiles/wood_wall_ss.png", "frameCount": 12, "frameSize": {"x": 16, "y": 16}}]`},
		{"frames that don't fit the image", `[` + debugEntry + `, {"tag": "wall", "path": "tiles/wood_wall_ss.png", "frameCount": 4, "frameSize": {"x": 24, "y": 16}}]`},
	} {
		if _, err := assetmanager.LoadAssetManagerFromJson(writeAssetList(t, c.list)); err == nil {
			t.Errorf("an asset list with %s was accepted", c.name)
		}
	}
}

func TestItemsCantShowFramesTheirImageDoesntHave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "items.json")
	for _, c := range []struct {
		name, item string
	}{
		{"an image that isn't in the list", `{"id": 1, "name": "X", "imageAsset": "nothing", "countable": true, "maxStack": 1}`},
		{"a frame past the image's last", `{"id": 1, "name": "X", "imageAsset": "torch_placed", "countable": true, "maxStack": 1, "animation": [{"frame": 3, "durationMs": 100}]}`},
	} {
		if err := os.WriteFile(path, []byte(`[`+c.item+`]`), 0o644); err != nil {
			t.Fatal(err)
		}
		atlas, err := ecs.LoadItemAtlasFromJson(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := atlas.CheckImageAssets(testutil.Assets(t)); err == nil {
			t.Errorf("an item with %s was accepted", c.name)
		}
	}
}
