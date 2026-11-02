package systems_test

import (
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"maps"
	"testing"
)

type openGround struct {
	damaged map[utils.Vec2i]int
}

func (openGround) IsOpaqueTile(utils.Vec2i) bool { return false }

func (openGround) TileLayer(utils.Vec2i) (ecs.LayerMask, bool) { return ecs.Layer_Ground, true }

func (g openGround) DamageTile(t utils.Vec2i, damage int) bool {
	g.damaged[t] += damage
	return true
}

func TestTheMattockStrikesTheTileUnderTheCursor(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	ground := openGround{damaged: map[utils.Vec2i]int{}}
	x.SetTerrain(ground)
	spawn := utils.TilePosToWorldPos(utils.Vec2i{X: 10, Y: 10})
	testutil.SetPos(t, x, player, spawn)

	cursor := spawn.Add(utils.Vec2f{X: 30, Y: -5})
	press(t, x, player, ecs.InputState{MainHandEqAbility1: 1, FacingDir: cursor})

	if want := map[utils.Vec2i]int{utils.WorldPosToTilePos(cursor): 50}; !maps.Equal(ground.damaged, want) {
		t.Fatalf("damaged %v, want %v", ground.damaged, want)
	}
}
