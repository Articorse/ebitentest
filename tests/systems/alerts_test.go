package systems_test

import (
	"ebittest/ecs/common"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

func TestAnAlertedEntityHasAnExclamationMarkOverItsSprite(t *testing.T) {
	x := testutil.NewECS(t)
	alerted := addSprite(t, x, utils.Vec2f{X: 100, Y: 100}, 1)
	addSprite(t, x, utils.Vec2f{X: 200, Y: 100}, 1)
	disabled := addSprite(t, x, utils.Vec2f{X: 300, Y: 100}, 1)
	x.SetDisabled(disabled, true)
	away := addSprite(t, x, utils.Vec2f{X: 5000, Y: 100}, 1)

	assets := testutil.Assets(t)
	marks, err := drawsystem.AlertMarks(
		drawsystem.DrawWindow(utils.Vec2f{}, utils.Vec2f{X: 640, Y: 360}),
		func(e common.EntityId) bool { return e == alerted || e == disabled || e == away },
		x, assets,
	)
	if err != nil {
		t.Fatal(err)
	}
	slime, err := assets.GetAsset(common.AssetSheetSlime)
	if err != nil {
		t.Fatal(err)
	}
	mark, err := assets.GetAsset(common.AssetImageExclamation)
	if err != nil {
		t.Fatal(err)
	}
	h, size := slime.Frames[0].Bounds().Dy(), mark.Frames[0].Bounds().Size()
	want := []utils.Vec2f{{X: 100 - float64(size.X)/2, Y: 100 - float64(h)/2 - 1 - float64(size.Y)}}
	if !slices.Equal(marks, want) {
		t.Fatalf("the marks are at %v, want only over the alerted entity in view, at %v", marks, want)
	}
}
