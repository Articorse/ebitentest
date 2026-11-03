package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"testing"
)

func TestEachFrameShowsForItsOwnDuration(t *testing.T) {
	x := testutil.NewECS(t)
	spr, err := ecs.NewSpriteComponent(common.AssetSheetSlime, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	ani, err := ecs.NewAnimationComponent(common.AssetSheetSlime, map[ecs.AnimationState][]ecs.AnimationFrame{
		ecs.Anim_Idle: {{FrameIdx: 0, DurationMs: 10 * data.TickMs}, {FrameIdx: 2, DurationMs: 2 * data.TickMs}, {FrameIdx: 1, DurationMs: 3 * data.TickMs}},
	})
	if err != nil {
		t.Fatal(err)
	}
	e := testutil.Add(t, x, spr, ani)
	s, _ := x.Sprites.Get(e)

	shown := []int{s.SubImageIdx()}
	for range 16 {
		if err := x.TickAnimation(e); err != nil {
			t.Fatal(err)
		}
		shown = append(shown, s.SubImageIdx())
	}

	want := []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 1, 1, 1, 0, 0}
	for i := range want {
		if shown[i] != want[i] {
			t.Fatalf("after each tick the sprite showed frames %v, want %v", shown, want)
		}
	}
}
