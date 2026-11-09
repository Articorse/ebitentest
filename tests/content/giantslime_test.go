package content_test

import (
	"ebittest/content/templates"
	"ebittest/ecs/systems/visionsystem"
	"ebittest/utils"
	"testing"
)

// The pond world with a wall down x = 3 from the top of the world to y = 0, open below it
type halfWallWorld struct {
	pondWorld
}

func (w halfWallWorld) wall(t utils.Vec2i) bool { return t.X == 3 && t.Y <= 0 }
func (w halfWallWorld) IsOpaqueTile(t utils.Vec2i) bool {
	return w.pondWorld.IsOpaqueTile(t) || w.wall(t)
}
func (w halfWallWorld) WalkSpeedPercent(t utils.Vec2i) int {
	if w.wall(t) {
		return 0
	}
	return w.pondWorld.WalkSpeedPercent(t)
}

func TestAGiantSlimeIsSeenWhereItShowsFromBehindAWall(t *testing.T) {
	x, player, slime := playerAndSlime(t, utils.TilePosToWorldPos(utils.Vec2i{X: -30, Y: -30}))
	x.SetTerrain(halfWallWorld{pondWorld{Terrain: x.Terrain()}})
	x.SetDisabled(slime, true)
	giant, err := templates.SpawnGiantSlime(x, utils.TilePosToWorldPos(utils.Vec2i{X: 5}))
	if err != nil {
		t.Fatal(err)
	}
	visionsystem.Tick(x)
	vision, _ := x.Visions.Get(player)
	if vision.CanSee(utils.Vec2i{X: 5}) {
		t.Fatal("the middle of the giant slime isn't behind the wall")
	}
	if seen, ok := x.NearestVisibleHostile(player); !ok || seen != giant {
		t.Fatal("the player doesn't see the giant slime where its body shows from behind the wall")
	}
}
