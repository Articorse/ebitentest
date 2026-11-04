package systems_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"strconv"
	"testing"
)

func addSprite(t *testing.T, x *ecs.ECSContainer, pos utils.Vec2f, layer uint8) common.EntityId {
	t.Helper()
	spr, err := ecs.NewSpriteComponent(common.AssetSheetSlime, layer, false)
	if err != nil {
		t.Fatal(err)
	}
	return testutil.AddMovable(t, x, pos, false, spr)
}

// The lowest opaque row of the image, which is where its sprite touches the ground
func bottomRow(t *testing.T, tag common.ImageAssetTag) float64 {
	t.Helper()
	t.Chdir(testutil.RepoRoot())
	asset, err := testutil.Assets(t).GetAsset(tag)
	if err != nil {
		t.Fatal(err)
	}
	return float64(asset.LayerYOffset)
}

// A tree whose trunk touches the ground at the given y. Its y in names is that ground y.
func tree(t *testing.T, groundY float64, layer uint8) common.StaticSprite {
	t.Helper()
	return common.StaticSprite{ImageAsset: common.AssetImageTree, Pos: utils.Vec2f{X: 50, Y: groundY - bottomRow(t, common.AssetImageTree)}, Layer: layer}
}

// What's drawn, in order: entity IDs, and trees by the y where they touch the ground
func drawOrder(t *testing.T, x *ecs.ECSContainer, statics ...common.StaticSprite) []string {
	t.Helper()
	t.Chdir(testutil.RepoRoot())
	treeBottom := bottomRow(t, common.AssetImageTree)
	order, err := drawsystem.SpriteDrawOrder(drawsystem.DrawWindow(utils.Vec2f{}, utils.Vec2f{X: 640, Y: 360}), statics, x, testutil.Assets(t))
	if err != nil {
		t.Fatal(err)
	}

	names := []string{}
	for _, d := range order {
		if d.Entity == -1 {
			names = append(names, "tree@"+strconv.Itoa(int(d.Static.Pos.Y+treeBottom)))
		} else {
			names = append(names, "e"+strconv.Itoa(int(d.Entity)))
		}
	}
	return names
}

func TestSpritesAndTreesAreDrawnFromTheBackToTheFront(t *testing.T) {
	x := testutil.NewECS(t)
	// The player touches the ground at y 100, whatever the art's empty margin below it
	playerY := 100 - bottomRow(t, common.AssetSheetSlime)
	player := addSprite(t, x, utils.Vec2f{X: 50, Y: playerY}, 20)
	held := addSprite(t, x, utils.Vec2f{X: 50, Y: playerY}, 20)
	testutil.Attach(t, x, held, player, ecs.Link_Held)
	north := addSprite(t, x, utils.Vec2f{X: 60, Y: -30}, 20)
	above := addSprite(t, x, utils.Vec2f{X: 50, Y: 0}, 25)
	hidden := addSprite(t, x, utils.Vec2f{X: 50, Y: 50}, 20)
	x.SetDisabled(hidden, true)
	addSprite(t, x, utils.Vec2f{X: 5000, Y: 50}, 20)

	got := drawOrder(t, x, tree(t, 110, 20), tree(t, 90, 20), tree(t, 100, 20), tree(t, 5000, 20))
	want := []string{
		"e" + strconv.Itoa(int(north)),
		"tree@90",
		"tree@100",
		"e" + strconv.Itoa(int(player)), "e" + strconv.Itoa(int(held)),
		"tree@110",
		"e" + strconv.Itoa(int(above)),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("drawn in order %v, want %v", got, want)
	}
}

func TestSpritesJustNorthOfTheOriginAreDrawnBehindThoseJustSouthOfIt(t *testing.T) {
	x := testutil.NewECS(t)
	// Touching the ground half a pixel either side of y 0, and added in this order, so that if both were counted as
	// row 0, the tie would put the south one first
	bottom := bottomRow(t, common.AssetSheetSlime)
	north := addSprite(t, x, utils.Vec2f{X: 50, Y: -0.5 - bottom}, 20)
	south := addSprite(t, x, utils.Vec2f{X: 50, Y: 0.5 - bottom}, 20)

	got := drawOrder(t, x)
	want := []string{"e" + strconv.Itoa(int(north)), "e" + strconv.Itoa(int(south))}
	if !slices.Equal(got, want) {
		t.Fatalf("drawn in order %v, want %v", got, want)
	}
}

func TestStaticSpritesOnTheirOwnAreDrawnFromTheBackToTheFront(t *testing.T) {
	t.Chdir(testutil.RepoRoot())
	stumpBottom := bottomRow(t, common.AssetImageTreeStump)
	stump := func(groundY float64, layer uint8) common.StaticSprite {
		return common.StaticSprite{ImageAsset: common.AssetImageTreeStump, Pos: utils.Vec2f{Y: groundY - stumpBottom}, Layer: layer}
	}

	front, behind, level, above := tree(t, 120, 20), stump(100, 20), tree(t, 100, 20), tree(t, 0, 25)
	got, err := drawsystem.StaticSpriteDrawOrder([]common.StaticSprite{above, front, level, behind}, testutil.Assets(t))
	if err != nil {
		t.Fatal(err)
	}

	// The tree and the stump touch the ground on the same row, so they keep the order they were given in
	if want := []common.StaticSprite{level, behind, front, above}; !slices.Equal(got, want) {
		t.Fatalf("drawn in order %+v, want %+v", got, want)
	}
}
