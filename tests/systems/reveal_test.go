package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/utils"
	"testing"
)

func TestOnlyWhatTheViewerSeesIsRevealed(t *testing.T) {
	wall := utils.Vec2i{X: 2}
	vision := ecs.NewVisionComponent(data.PlayerVisionRangeTiles, 1)
	vision.Update(utils.Vec2i{}, data.PlayerVisionRangeTiles, func(t utils.Vec2i) bool { return t == wall }, func(utils.Vec2i) uint8 { return data.NightSkyLight })

	for _, c := range []struct {
		name string
		pos  utils.Vec2f
		want bool
	}{
		{"the viewer's tile", utils.TilePosToWorldPos(utils.Vec2i{}), true},
		{"a seen tile", utils.TilePosToWorldPos(utils.Vec2i{Y: 3}).Add(utils.Vec2f{X: 7}), true},
		{"behind the wall", utils.TilePosToWorldPos(utils.Vec2i{X: 4}), false},
		{"too dim for how far it is", utils.TilePosToWorldPos(utils.Vec2i{Y: -20}), false},
	} {
		if got := drawsystem.Revealed(c.pos, vision); got != c.want {
			t.Errorf("%s: revealed %v, want %v", c.name, got, c.want)
		}
	}
}
