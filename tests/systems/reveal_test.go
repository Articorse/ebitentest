package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/utils"
	"testing"
)

func TestOnlyWhatTheVisionOverlayUncoversIsRevealed(t *testing.T) {
	wall := utils.Vec2i{X: 2}
	vision := ecs.NewVisionComponent(8)
	vision.Update(utils.Vec2i{}, float64(vision.Radius()), func(t utils.Vec2i) bool { return t == wall })
	center := utils.TilePosToWorldPos(utils.Vec2i{})
	_, outer := vision.CircleRadii()

	for _, c := range []struct {
		name string
		pos  utils.Vec2f
		want bool
	}{
		{"the viewer's tile", center, true},
		{"a seen tile in the circle", utils.TilePosToWorldPos(utils.Vec2i{Y: 3}), true},
		{"behind the wall", utils.TilePosToWorldPos(utils.Vec2i{X: 4}), false},
		{"a seen tile past the circle", center.Add(utils.Vec2f{Y: -outer - data.TileSize/4}), false},
	} {
		if got := drawsystem.Revealed(c.pos, center, vision); got != c.want {
			t.Errorf("%s: revealed %v, want %v", c.name, got, c.want)
		}
	}
}
